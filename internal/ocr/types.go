package ocr

type Line struct {
	Text   string `json:"Text"`
	X      int    `json:"X"`
	Y      int    `json:"Y"`
	Width  int    `json:"Width"`
	Height int    `json:"Height"`
}
