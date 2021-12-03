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
    "c", "/searcher/config/searcher.jsonc", "The searcher configuration file",
  )
  webServerHost := flag.String(
  	"H", "", "The interface on which the webServer will listen",
  )
  webServerPort := flag.Int(
  	"p", 0, "The port on which the webServer will listen",
  )
  logFilePath := flag.String(
    "l", "stderr", "The searcher log file path",
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

  go runWebServer(*webServerHost, int64(*webServerPort))

  indexFiles()

}


