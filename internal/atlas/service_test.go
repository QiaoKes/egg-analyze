package atlas

import (
	"encoding/json"
	"testing"
)

func TestNormalizePetID(t *testing.T) {
	if got := normalizePetID("2"); got != "002" {
		t.Fatalf("unexpected normalized id: got %q want %q", got, "002")
	}
	if got := normalizePetID("011"); got != "011" {
		t.Fatalf("unexpected normalized id: got %q want %q", got, "011")
	}
}

func TestAssetURL(t *testing.T) {
	want := "https://rocom.mfsky.qzz.io/creature-atlas/162-base.webp"
	if got := assetURL("162-base.webp"); got != want {
		t.Fatalf("unexpected asset url: got %q want %q", got, want)
	}
}

func TestFetchIndexUsesDefaultImageFromMasterList(t *testing.T) {
	payload := `{
		"creatures": [
			{"id": "012", "images": {"default": "012-form-01.webp"}},
			{"id": "162", "images": {"default": "162-base.webp"}},
			{"id": " 2 ", "images": {"default": "002-base.webp"}}
		]
	}`

	var list masterList
	if err := json.Unmarshal([]byte(payload), &list); err != nil {
		t.Fatalf("unmarshal master list: %v", err)
	}

	index := make(map[string]string, len(list.Creatures))
	for _, item := range list.Creatures {
		index[normalizePetID(item.ID)] = item.Images.Default
	}

	if got := index["012"]; got != "012-form-01.webp" {
		t.Fatalf("unexpected default image for 012: %q", got)
	}
	if got := index["162"]; got != "162-base.webp" {
		t.Fatalf("unexpected default image for 162: %q", got)
	}
	if got := index["002"]; got != "002-base.webp" {
		t.Fatalf("unexpected normalized image for 002: %q", got)
	}
}
