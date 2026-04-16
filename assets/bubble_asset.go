package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed bubble.png
var BubblePNG []byte

var BubbleResource = fyne.NewStaticResource("bubble.png", BubblePNG)
