package main

// see: https://bogotobogo.com/GoLang/GoLang_SQLite.php

import (
  "os"
  "log"
  "time"
  "flag"
  "math/rand"
)

func main() {
  configFilePath := flag.String(
    "c", "searcher.jsonc", "The searcher configuration file",
  )
  logFilePath := flag.String(
    "l", "stderr", "The searcjer log file path",
  )

  flag.Parse()

  // Setup logging
  switch *logFilePath {
    case "stdout" : log.SetOutput(os.Stdout)
    case "stderr" : // nothing to do...
    case ""       : // nothing to do (use stderr)...
    default       :
      logFile, err := os.OpenFile(
        *logFilePath ,
        os.O_CREATE|os.O_APPEND|os.O_WRONLY ,
        0644 ,
      )
      if err != nil { log.Fatal(err) }
      defer logFile.Close()
      log.SetOutput(logFile)
  }
  log.Print("Searcher: starting");
  defer log.Print("Searcher: finished");

  // load the configuration file
  setConfigFilePath(*configFilePath)

  // setup sleep's random number generator
  rand.Seed(time.Now().UnixNano())

  // ensure the database exists and has the structure we require
  initDatabaseStructure()

  go runWebServer()

  indexFiles()

}


