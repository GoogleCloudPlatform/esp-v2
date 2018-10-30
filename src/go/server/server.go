package main

import (
  "fmt"

  "cloudesf.googleresource.com/gcpproxy/src/go/configmanager"
)

func main() {
  // TODO(jilinxia): pass in service name.
  m, _ := configmanager.NewConfigManager("library-example.googleapis.com")
  err := m.Init("2017-05-01r0")
  if err != nil {
    fmt.Errorf("fail to FetchRollouts")
  }
}
