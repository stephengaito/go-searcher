package main

// see: https://bogotobogo.com/GoLang/GoLang_SQLite.php

import (
  "os"
  "log"
  "strings"
  "strconv"
  "net/url"
  "net/http"
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
)

func WebserverMaybeFatal(logMessage string, err error) {
  if err != nil {
    log.Fatalf("Webserver(FATAL): %s ERROR: %s", logMessage, err)
  }
}

func WebserverMaybeError(logMessage string, err error) {
  if err != nil {
    log.Printf("Webserver(error): %s error: %s",logMessage, err)
  }
}

func WebserverLog(logMesg string) {
  log.Printf("Webserver(info): %s", logMesg)
}

func WebserverLogf(logFormat string, v ...interface{}) {
  log.Printf("Webserver(info): "+logFormat, v...)
}

type SearchResults struct {
  FilePath string
  Title    string
  Type     string
  Rank     string
}

type SearchData struct {
  Query       string
  MaxNum      int
  MaxNumRange []int
  Results     []SearchResults
}

func runWebServer() {

  htmlDirs := getConfigAStr("HtmlDirs", []string{ "files" })
  urlBase  := getConfigStr("UrlBase", "")

  searchForm := CreateTemplate(
    getConfigStr("Webserver.SearchForm", "config/searchForm.html"),
  )

  searchDB, err := sql.Open("sqlite3", getConfigStr("DatabasePath", ""))
  WebserverMaybeFatal("trying to open the database", err)
  defer searchDB.Close()

  http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
    // do not do anything!
  })

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    WebserverLogf("url: [%s]", r.URL.Path)
    userQuery  := ""
    maxNum := int(getConfigInt("Webserver.MaxNumResults", 100))
    if r.Method == http.MethodGet {
      userQuery = strings.Replace(r.URL.Path, "/search/", "", 1)
      newQuery, err := url.QueryUnescape(userQuery)
      if err == nil { userQuery = newQuery  }
    } else if r.Method == http.MethodPost {
      r.ParseForm()
      userQuery  = r.Form.Get("searchQueryStr")
      tmpMaxNum, err := strconv.Atoi(r.Form.Get("searchQueryNum"))
      if err != nil { tmpMaxNum = maxNum }
      maxNum = tmpMaxNum
    }

    sqlQuery := strings.Replace(userQuery, "'", "''", -1)
    WebserverLogf("query: [%s]", sqlQuery)
    var searchData SearchData
    searchData.Query  = userQuery
    searchData.MaxNum = maxNum
    searchData.MaxNumRange = []int{10, 50, 100, 200}
    if 0 < len(sqlQuery) {
      results := make([]SearchResults, maxNum)
      sqlCmd := "select filePath, fileTitle, bm25(pageSearch) from pageSearch('"+sqlQuery+"') order by rank;"
      WebserverLogf("sqlCmdQuery: [%s]", sqlCmd)
      rows, err := searchDB.Query(sqlCmd)
      WebserverMaybeError("trying to search pageSearch table with query", err)
      defer rows.Close()
      //
      numResults := 0
      for {
        if maxNum <= numResults { break }
        hasRow := rows.Next()
        if  !hasRow {
          err := rows.Err()
          WebserverMaybeError("could not step into next row", err)
          break
        }
        var filePath string
        var title    string
        var rank     float64
        err = rows.Scan(&filePath, &title, &rank)
        WebserverMaybeError("scanning filePath and title from results", err)
        if _, err = os.Stat(filePath); err != nil { continue }
        for _, anHtmlDir := range htmlDirs {
          if strings.HasPrefix(filePath, anHtmlDir) {
            results[numResults].FilePath =
              strings.Replace(filePath, anHtmlDir, urlBase, 1)
          }
        }
        results[numResults].Title    = title
        results[numResults].Rank     = strconv.FormatFloat(-1 * rank, 'f', 2, 64)
        var resultType string
        switch {
          case strings.Contains(filePath, "blog")   : resultType = "B"
          case strings.Contains(filePath, "author") : resultType = "A"
          case strings.Contains(filePath, "cite")   : resultType = "C"
          case strings.Contains(filePath, "tasks")  : resultType = "T"
          default : resultType = " "
        }
        results[numResults].Type = resultType
        numResults = numResults + 1
      }
      rows.Close()
      searchData.Results = results[:numResults]
    } else {
      searchData.Results = []SearchResults{}
    }

    err = searchForm.execute(w, searchData )
    WebserverMaybeError("could not execute searchForm", err)
  })

  http.ListenAndServe(":8080", nil)
}
