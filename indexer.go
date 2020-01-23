package main

// For rss/Atom feeds use: https://github.com/mmcdole/gofeed


import (
  "os"
  "log"
  "time"
  "io/ioutil"
  "regexp"
  "strings"
  "path/filepath"
  "math/rand"
  "github.com/bvinc/go-sqlite-lite/sqlite3"
  "github.com/grokify/html-strip-tags-go"
)

func IndexerMaybeFatal(logMessage string, err error) {
  if err != nil {
    log.Fatalf("Indexer(FATAL): %s ERROR: %s", logMessage, err)
  }
}

func IndexerMaybeError(logMessage string, err error) {
  if err != nil {
    log.Printf("Indexer(error): %s error: %s",logMessage, err)
  }
}

func IndexerLog(logMesg string) {
  log.Printf("Indexer(info): %s", logMesg)
}

func IndexerLogf(logFormat string, v ...interface{}) {
  log.Printf("Indexer(info): "+logFormat, v...)
}

// Conditionally initialize the database structure
//
func initDatabaseStructure() {
  //
  // Only initialize the database if it does not already exist...
  //
  if _, err := os.Stat(getConfigStr("DatabasePath", "")); os.IsNotExist(err) {
    //
    // If it does not exist... try to create it...
    //
    if searcherFile, err := os.Create(getConfigStr("DatabasePath", ""));  err != nil {
      IndexerMaybeFatal("could not create database file", err)
    } else {
      searcherFile.Close()
    }
    //
    // We have been able to create the database so...
    // ... create the tables we need...
    //
    searchDB, err := sqlite3.Open(getConfigStr("DatabasePath", ""))
    IndexerMaybeFatal("could not open database file to initialize tables", err)
    defer searchDB.Close()

    err = searchDB.Exec(`
      create table fileInfo (
        filePath  text not null primary key,
        fileMTime int,
        fileSize  int
      );
    `)
    IndexerMaybeFatal("could not create fileInfo table", err)

    err = searchDB.Exec(`
      create index filePaths ON fileInfo(filePath);
    `)
    IndexerMaybeFatal("could not create filePaths index", err)

    err = searchDB.Exec(`
      create virtual table pageSearch using fts5(
        filePath,
        fileTitle,
        fileStr
      );
    `)
    IndexerMaybeFatal("could not create pageSearch table", err)
  }
}

func removeMissingFiles(searchDB *sqlite3.Conn) {
  maxDeletions := getConfigInt("Indexer.RemoveBatch", 200)
  numDeletions := int64(0)
  var filesToDelete []string = make([]string, maxDeletions)

  IndexerLog("removing missing files")
  //
  // look for files which are in fileInfo database...
  // ... but no longer exist... (store them for later deletion)
  //
  rows, err := searchDB.Prepare("select filePath from fileInfo")
  IndexerMaybeError("selecting filePaths from fileInfo", err)
  defer rows.Close()
  //
  for { 
    if maxDeletions <= numDeletions { break }
    //
    hasRow, err := rows.Step()
    IndexerMaybeError("trying to step to next row", err)
    if !hasRow { break }
    //
    var fullPath string
    err = rows.Scan(&fullPath)
    IndexerMaybeError("scaning filePath from results", err)
    if _, err := os.Stat(fullPath); err == nil { continue }
    filesToDelete[numDeletions] = fullPath
    numDeletions = numDeletions + 1
  }
  rows.Close()

  // Now acutally delete the files from the database
  //
  for i := int64(0); i < numDeletions; i++ {
    aFile := filesToDelete[i]
    IndexerLogf("deleting(%d): [%s]", i, aFile)
    err = searchDB.Begin()
    if err != nil {
      IndexerMaybeError("could not start deletion transaction", err)
      break
    }
    err = searchDB.Exec("delete from fileInfo where filePath = ?", aFile)
    if err != nil {
      IndexerMaybeError("deleting from fileInfo", err)
      searchDB.Rollback()
      break
    }
    err = searchDB.Exec("delete from pageSearch where filePath = ?", aFile)
    if err != nil {
      IndexerMaybeError("deleting from pageSearch", err)
      searchDB.Rollback()
      break
    }
    err = searchDB.Commit()
    if err != nil {
      IndexerMaybeError("could not commit deletion transaction", err)
      break
    }
  }

  // Now shrink the database by vacuuming it...
  //
  if 0 < numDeletions {
    IndexerLog("vacuuming database....")
    searchDB.Exec("vacuum;")
    IndexerLog("finished vacuuming database.")
  }
  IndexerLogf("removed %d missing files", numDeletions)
}

func lookForNewFiles(searchDB *sqlite3.Conn) {
  maxInsertions := getConfigInt("Indexer.AddUpdateBatch", 200)
  numInsertions := int64(0)
  titleRegexp   := regexp.MustCompile(`<meta name="pageTitle" content="(.*?)">`)

  IndexerLog("looking for new or chagned files")
  //
  // walk the html files looking for new or changed files...
  //
  htmlDirs := getConfigAStr("HtmlDir", ["html"])
  for anHtmlDir := range htmlDir {
    filepath.Walk(anHtmlDir,func (path string, info os.FileInfo, err error) error {
      if maxInsertions <= numInsertions {
        return nil
      }
      if err != nil {
        IndexerMaybeError("walking path "+path, err)
        return nil
      }
      if info.IsDir() {
//        IndexerLogf("walking into directory %s", path)
        return nil
      }
      if !strings.HasSuffix(path, ".html") { return nil }
      if strings.HasSuffix(path, "index.html") { return nil }
      if strings.HasSuffix(path, "Citations.html") { return nil }
      var filePath  string = ""
      var pageMTime int64  = 0
      var pageSize  int64  = 0
      rows, err := searchDB.Prepare(`
        select * from fileInfo where filePath == ? ;
      `, path)
      IndexerMaybeError("looking for new files in fileInfo", err)
      hasRows, err := rows.Step()
      IndexerMaybeError("looking for first result from files in fileInfo", err)
      if hasRows {
        rows.Scan(&filePath, &pageMTime, &pageSize)
      }
      rows.Close()
      //
      fileInfo, _ := os.Stat(path)
      if fileInfo.ModTime().Unix() == pageMTime && fileInfo.Size() == pageSize {
        return nil
      }

      IndexerLogf("need to index(%d) [%s]", numInsertions+1, path)
      //
      // start by getting the values for the file itself
      //
      fileBytes, _     := ioutil.ReadFile(path)
      fileStr          := string(fileBytes)
      fileStr           = strings.Replace(fileStr, "\n", " ", -1)
      fileStr           = strings.Replace(fileStr, "\r", " ", -1)
      fileTitleMatches := titleRegexp.FindStringSubmatch(fileStr)
      // The following is a dirty hack to protect us from missing titles ;-(
      fileTitle        := path
      //IndexerLogf("titleMatches [%s]", fileTitleMatches)
      if 0 < len(fileTitleMatches) {
        fileTitle = string(fileTitleMatches[1])
      }
      //IndexerLogf("title [%s]", fileTitle)
      //
      fileStr = strip.StripTags(fileStr)
      removeSpaces, _ := regexp.Compile(`\s+`)
      fileStr = removeSpaces.ReplaceAllString(fileStr, " ")
      //
      // now check if there is an associated *Citations.html file....
      //   (this is a hack for the current Jekyll bases references system)
      //
      citationsPath := strings.Replace(path, ".html", "Citations.html", 1)
      citationsFileBytes, err := ioutil.ReadFile(citationsPath)
      if err == nil {
        citationsFileStr := string(citationsFileBytes)
        citationsFileStr  = strip.StripTags(citationsFileStr)
        citationsFileStr  = removeSpaces.ReplaceAllString(citationsFileStr, " ")
        fileStr = fileStr + " " + citationsFileStr
      }

      if filePath != path {
        //
        // this file has not yet been indexed... so insert it...
        //
        IndexerLogf("INSERTING: [%s][%s]", path, fileTitle)
        err := searchDB.Begin()
        if err != nil {
          IndexerMaybeError("could not start insertions transaction", err)
          return nil
        }
        err = searchDB.Exec(`
          insert into fileInfo ( filePath, fileMTime, fileSize ) values ( ?, ?, ?)
        `, path, fileInfo.ModTime().Unix(), fileInfo.Size())
        if err != nil {
          IndexerMaybeError("trying to insert new file into fileInfo", err)
          searchDB.Rollback()
          return nil
        }
        err = searchDB.Exec(`
          insert into pageSearch ( filePath, fileTitle, fileStr ) values ( ?, ?, ?)
        `, path, fileTitle, fileStr)
        if err != nil {
          IndexerMaybeError("trying to insert new file into pageSearch", err)
          searchDB.Rollback()
          return nil
        }
        err = searchDB.Commit()
        if err != nil {
          IndexerMaybeError("could not commit insertions transaction", err)
          return nil
        }
        numInsertions = numInsertions + 1
        return nil
      }

      //
      // this file has already been indexed... so update it...
      //
      IndexerLogf("UPDATING: [%s][%s]", path, fileTitle)
      err = searchDB.Begin()
      if err != nil {
        IndexerMaybeError("could not start update transaction", err)
        return nil
      }
      err = searchDB.Exec(`
        update fileInfo set fileMTime = ?, fileSize = ? where filePath = ?
      `, fileInfo.ModTime().Unix(), fileInfo.Size(), path)
      if err != nil {
        IndexerMaybeError("trying to update changed file into fileInfo", err)
        searchDB.Rollback()
        return nil
      }
      err = searchDB.Exec(`
        update pageSearch set fileTitle = ?, fileStr = ? where filePath = ?
      `, fileTitle, fileStr, path)
      if err != nil {
        IndexerMaybeError("trying to update changed file into pageSearch", err)
        searchDB.Rollback()
        return nil
      }
      err = searchDB.Commit()
      if err != nil {
        IndexerMaybeError("could not commit update transaction", err)
        return nil
      }
      numInsertions = numInsertions + 1
      return nil
    })
  }
  IndexerLogf("Indexer: found %d new or changed files", numInsertions)
}

func indexFiles() {
  //
  // Begin by opening the database
  //
  searchDB, err := sqlite3.Open(getConfigStr("DatabasePath", ""))
  IndexerMaybeFatal("could not open database", err)
  defer searchDB.Close()
  //
  // Now periodically scan the file system for new pages
  //
  for {
    IndexerLog("starting");
    removeMissingFiles(searchDB)
    lookForNewFiles(searchDB)
    IndexerLog("finished");
    sleepSeconds := getConfigInt("Indexer.SleepSeconds", 60)
    time.Sleep(time.Duration(rand.Int63n(sleepSeconds)) * time.Second)
  }
}
