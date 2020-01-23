package main

/*

  We use the https://github.com/tidwall/gjson and
  https://github.com/tidwall/sjson packages to get/set the raw json.
  Very importantly, both packages ignore golang comments embedded in the
  json!

  We implement a lazy loading of the config file so that we can allow for
  the config file to be changed outside the searcher. TO DO THIS, we
  require a RWMutex. SEE: https://stackoverflow.com/a/19168242 for a good
  discussion on how to USE RWMutex's.

*/

import (
  "os"
  "log"
  "time"
  "sync"
  "io/ioutil"
  "github.com/tidwall/gjson"
  "github.com/tidwall/sjson"
)

var configFilePath  string = ""
var configFileMTime int64  = 0
var configFileSize  int64  = 0
var searcherConfig  string = ""
var updateConfig    sync.RWMutex

/////////////////////////////
// Locking primitives
//
// Generally kept short and simple!
// USE defer's
// USE RLock's if nothing is being changed
// USE Lock's if something might be changed

func setConfigFilePath(aConfigFilePath string) {
  updateConfig.Lock()
  defer updateConfig.Unlock()
  configFilePath = aConfigFilePath
}

func hasConfigChanged() bool {
  updateConfig.RLock()
  defer updateConfig.RUnlock()

  if len(searcherConfig) < 1 {
    //log.Print("Config(reload): no configuration data")
    return true
  }

  if len(configFilePath) < 1 {
    log.Print("Config(no reload): no configuration path")
    return false
  }

  configFileInfo, err := os.Stat(configFilePath)
  if err != nil {
    log.Printf(
      "Config(no reload): could not get file info for [%s] ERROR: %s",
       configFilePath, err,
    )
    return false
  }

  if configFileMTime != configFileInfo.ModTime().Unix() ||
     configFileSize  != configFileInfo.Size() {
    log.Printf("Config(reload): file info changed: MTime: %d(%d) Size: %d(%d)" ,
      configFileInfo.ModTime().Unix(), configFileMTime ,
      configFileInfo.Size(), configFileSize ,
    )
    return true
  }

//  log.Print("Config(no reload)")
  return false
}

func reloadConfigFile() {
  updateConfig.Lock()
  defer updateConfig.Unlock()

  searcherConfigBytes, err := ioutil.ReadFile(configFilePath)
  if err == nil {
    searcherConfig = string(searcherConfigBytes)
    configFileInfo, err := os.Stat(configFilePath)
    if err == nil {
      configFileMTime = configFileInfo.ModTime().Unix()
      configFileSize  = configFileInfo.Size()
    }
  } else {
    log.Printf(
      "Searcher(error): Failed to load configuration from [%s] ERROR: %s",
      configFilePath, err,
    )
  }

  // Now ensure we have the basic REQUIRED configuraiton...
  //
  gValue := gjson.Get(searcherConfig, "DatabasePath")
  if ! gValue.Exists() {
    tmpSearcherConfig, err := sjson.Set(
      searcherConfig, "DatabasePath", "data/searcher.db",
    )
    if err == nil {
      searcherConfig = tmpSearcherConfig
    } else {
      log.Fatalf(
"Searcher(fatal): Failed to SET configuration value DatabasePath when required ERROR: %s",
        err,
      )
    }
  }
  gValue = gjson.Get(searcherConfig, "HtmlDirs")
  if ! gValue.Exists() {
    tmpSearcherConfig, err := sjson.Set(
      searcherConfig, "HtmlDirs", "[ \"files\" ]",
    )
    if err == nil {
      searcherConfig = tmpSearcherConfig
    } else {
      log.Fatalf(
"Searcher(fatal): Failed to SET configuration value HtmlDirs when required ERROR: %s",
        err,
      )
    }
  }
  log.Printf(
    "Searcher: configuration: [%s]\n[%s]", configFilePath, searcherConfig,
  )
}

func getConfigVar(configVarPath string) gjson.Result {
  if hasConfigChanged() { reloadConfigFile() }

  updateConfig.RLock()
  defer updateConfig.RUnlock()
  return gjson.Get(searcherConfig, configVarPath)
}


///////////////////////////
// Non locking primitives
//
// Which depend upon the locking primitives above
//

func getConfigStr(configVarPath string, aDefault string) string {
  gValue := getConfigVar(configVarPath)
  theValue := aDefault
  if gValue.Exists() { theValue = gValue.String()  }
  return theValue
}

func getConfigAStr(configVarPath string, aDefault []string) []string {
  gValue := getConfigVar(configVarPath)
  theValue := aDefault
  if gValue.Exists() {
    someValues = gValue.Array()
    theValue = make([]string, 0)
    for _, aStrValue := range someValues {
      theValue = append(theValue, aStrValue.String())
    }
  }
  return theValue
}

func getConfigBool(configVarPath string, aDefault bool) bool {
  gValue := getConfigVar(configVarPath)
  theValue := aDefault
  if gValue.Exists() { theValue = gValue.Bool()  }
  return theValue
}

func getConfigInt(configVarPath string, aDefault int64) int64 {
  gValue := getConfigVar(configVarPath)
  theValue := aDefault
  if gValue.Exists() { theValue = gValue.Int()  }
  return theValue
}

func getConfigUInt(configVarPath string, aDefault uint64 ) uint64 {
  gValue := getConfigVar(configVarPath)
  theValue := aDefault
  if gValue.Exists() { theValue = gValue.Uint()  }
  return theValue
}

func getConfigFloat(configVarPath string, aDefault float64) float64 {
  gValue := getConfigVar(configVarPath)
  theValue := aDefault
  if gValue.Exists() { theValue = gValue.Float()  }
  return theValue
}

func getConfigTime(configVarPath string, aDefault time.Time) time.Time {
  gValue := getConfigVar(configVarPath)
  theValue := aDefault
  if gValue.Exists() { theValue = gValue.Time()  }
  return theValue
}
