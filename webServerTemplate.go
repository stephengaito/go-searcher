package main

/*

  We implement a lazy reloading of the template file so that we can allow
  for the template file to be changed outside the searcher. TO DO THIS, we
  require a RWMutex. SEE: https://stackoverflow.com/a/19168242 for a good
  discussion on how to USE RWMutex's.

*/


import (
  "io"
  "os"
  "log"
  "sync"
  "html/template"
)

type webServerTemplate struct {
  filePath  string
  fileMTime int64
  fileSize  int64
  template  *template.Template
  update    sync.RWMutex
}

func CreateTemplate(aTemplatePath string) *webServerTemplate {
  if len(aTemplatePath) < 1 {
    log.Fatal("WebserverTemplate: no template path supplied!")
  }

  wst := new(webServerTemplate)
  wst.filePath  = aTemplatePath
  wst.fileMTime = 0
  wst.fileSize  = 0
  newTemplate, err := template.ParseFiles(aTemplatePath)
  if err != nil {
    log.Fatal(
      "WebserverTemplate: could not load initial template [%s] ERROR: %s",
      aTemplatePath, err,
    )
  }
  wst.template = newTemplate

  templateFileInfo, err := os.Stat(wst.filePath)
  if err == nil {
    wst.fileMTime = templateFileInfo.ModTime().Unix()
    wst.fileSize  = templateFileInfo.Size()
  }

  log.Printf("WebserverTemplate: loaded [%s]", wst.filePath)

  return wst
}

func (wst *webServerTemplate) hasTemplateChanged() bool {
  wst.update.RLock()
  defer wst.update.RUnlock()

  templateFileInfo, err := os.Stat(wst.filePath)
  if err != nil {
    log.Printf(
"WebserverTemplate(no reload): could not get template file info for [%s] ERROR: %s",
       wst.filePath, err,
    )
    return false
  }

  if wst.fileMTime != templateFileInfo.ModTime().Unix() ||
     wst.fileSize  != templateFileInfo.Size() {
    log.Printf(
      "WebserverTemplate(reload): file info changed: MTime: %d(%d) Size: %d(%d)",
       templateFileInfo.ModTime().Unix(), wst.fileMTime,
       templateFileInfo.Size(), wst.fileSize,
    )
    return true
  }
//  log.Print("WebserverTemplate(no reload)")
  return false
}

func (wst *webServerTemplate) reloadTemplate() {
  wst.update.Lock()
  defer wst.update.Unlock()

  newTemplate, err := template.ParseFiles(wst.filePath)
  if err == nil {
    wst.template = newTemplate
    templateFileInfo, err := os.Stat(wst.filePath)
    if err == nil {
      wst.fileMTime = templateFileInfo.ModTime().Unix()
      wst.fileSize  = templateFileInfo.Size()
    }
  } else {
    log.Printf(
      "WebserverTemplate: failed to load template from [%s] ERROR: %s",
      wst.filePath, err,
    )
  }
}

func (wst *webServerTemplate) execute(wr io.Writer, data interface{}) error {
  if wst.hasTemplateChanged() { wst.reloadTemplate() }

  wst.update.RLock()
  defer wst.update.RUnlock()

  return wst.template.Execute(wr, data)
}
