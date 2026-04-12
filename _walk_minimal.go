package main

import (
  "github.com/lxn/walk"
)

func main() {
  mw, err := walk.NewMainWindow()
  if err != nil {
    panic(err)
  }
  mw.Run()
}
