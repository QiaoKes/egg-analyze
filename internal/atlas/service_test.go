package atlas

import "testing"

func TestNormalizeSpriteKey(t *testing.T) {
	if got := normalizeSpriteKey(" JL_duoling.webp "); got != "duoling" {
		t.Fatalf("unexpected normalized key: got %q want %q", got, "duoling")
	}
	if got := normalizeSpriteKey("duoduo"); got != "duoduo" {
		t.Fatalf("unexpected raw key: got %q want %q", got, "duoduo")
	}
}

func TestAssetURL(t *testing.T) {
	want := "https://rocom.aoe.top/assets/webp/friends/JL_duoling.webp"
	if got := assetURL("duoling"); got != want {
		t.Fatalf("unexpected asset url: got %q want %q", got, want)
	}
}
