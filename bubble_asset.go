package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/bubble.png
var bubblePNG []byte

var bubbleImageResource = fyne.NewStaticResource("bubble.png", bubblePNG)
