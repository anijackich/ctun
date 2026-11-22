package trayicon

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func alphaAt(t *testing.T, data []byte, x, y int) uint8 {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds() != image.Rect(0, 0, size, size) {
		t.Fatalf("bounds %v", img.Bounds())
	}
	_, _, _, a := img.At(x, y).RGBA()
	return uint8(a >> 8)
}

func TestLitBarsAreOpaque(t *testing.T) {
	g := Connected(1)
	firstBar, lastBar := 13, 48
	if a := alphaAt(t, g.Black, firstBar, 50); a != 0xff {
		t.Errorf("lit bar alpha %d", a)
	}
	if a := alphaAt(t, g.Black, lastBar, 20); a == 0 || a > 0x60 {
		t.Errorf("unlit bar alpha %d", a)
	}
	if a := alphaAt(t, g.White, 2, 2); a != 0 {
		t.Errorf("background alpha %d", a)
	}
}

func TestDisconnectedIsFaint(t *testing.T) {
	if a := alphaAt(t, Disconnected().Black, 13, 50); a == 0 || a > 0x70 {
		t.Errorf("alpha %d", a)
	}
}

func TestLogoHasAnEye(t *testing.T) {
	logo := Logo().Black
	if a := alphaAt(t, logo, 27, 32); a != 0xff {
		t.Errorf("body alpha %d", a)
	}
	if a := alphaAt(t, logo, 17, 30); a != 0 {
		t.Errorf("eye alpha %d", a)
	}
}
