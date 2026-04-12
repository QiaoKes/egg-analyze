package ocr

func New() Recognizer {
	recognizer, err := newRapidOCRRecognizer()
	if err == nil {
		return recognizer
	}
	return &unavailableRecognizer{message: err.Error()}
}
