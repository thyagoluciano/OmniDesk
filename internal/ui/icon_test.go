package ui

import (
	"bytes"
	"image"
	_ "image/png"
	"testing"

	"omnidesk/assets"
)

func TestGeneratedIcon(t *testing.T) {
	b := GenerateIconBytes()
	if len(b) == 0 {
		t.Fatal("empty generated icon bytes")
	}
	img, format, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to decode generated icon: %v", err)
	}
	if format != "png" {
		t.Fatalf("expected png format, got %s", format)
	}
	if img.Bounds().Dx() != 64 || img.Bounds().Dy() != 64 {
		t.Fatalf("expected 64x64, got %v", img.Bounds())
	}
}

func TestAssetsIconPNG(t *testing.T) {
	if len(assets.IconPNG) == 0 {
		t.Fatal("empty assets.IconPNG")
	}
	img, format, err := image.Decode(bytes.NewReader(assets.IconPNG))
	if err != nil {
		t.Fatalf("failed to decode assets.IconPNG: %v", err)
	}
	if format != "png" {
		t.Fatalf("expected png format, got %s", format)
	}
	t.Logf("assets.IconPNG bounds: %v", img.Bounds())
}

func TestGetTrayIcon(t *testing.T) {
	b := GetTrayIcon()
	if len(b) == 0 {
		t.Fatal("empty GetTrayIcon bytes")
	}
}
