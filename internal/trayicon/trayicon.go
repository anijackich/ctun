package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/vector"
)

const Bars = 4

const (
	size       = 64
	unit       = size / 24.0
	unlitAlpha = 0.3
	offAlpha   = 0.38
	// kappa places cubic control points so that a quarter turn approximates a circle.
	kappa = 0.5523
)

var barHeights = [Bars]float32{0.35, 0.55, 0.78, 1}

type Glyph struct {
	Black, White []byte
}

func Connected(lit int) Glyph {
	return signal(func(bar int) float64 {
		if bar < lit {
			return 1
		}
		return unlitAlpha
	})
}

func Disconnected() Glyph {
	return signal(func(int) float64 { return offAlpha })
}

func Logo() Glyph {
	mask := image.NewAlpha(image.Rect(0, 0, size, size))
	fill(mask, 1, func(p *pen) {
		p.move(2, 12.2)
		p.cube(4.8, 7.6, 10.5, 6.6, 15.6, 10.3)
		p.line(21.6, 5.6)
		p.quad(19.1, 12, 21.6, 18.4)
		p.line(15.6, 13.9)
		p.cube(10.5, 17.6, 4.8, 16.8, 2, 12.2)
		p.close()
		p.move(9.2, 8.3)
		p.line(12.4, 4.9)
		p.line(13.3, 9)
		p.close()
		p.circle(6.6, 11.55, 1.15, true)
	})
	return encode(mask)
}

func signal(alpha func(bar int) float64) Glyph {
	const width, gap, radius, bottom, span = 3, 1.7, 1.1, 20, 16
	mask := image.NewAlpha(image.Rect(0, 0, size, size))
	left := float32(24-(Bars*width+(Bars-1)*gap)) / 2
	for i, h := range barHeights {
		x := left + float32(i)*(width+gap)
		fill(mask, alpha(i), func(p *pen) { p.roundRect(x, bottom-span*h, x+width, bottom, radius) })
	}
	return encode(mask)
}

type pen struct{ r *vector.Rasterizer }

func (p pen) move(x, y float32) { p.r.MoveTo(x*unit, y*unit) }
func (p pen) line(x, y float32) { p.r.LineTo(x*unit, y*unit) }
func (p pen) close()            { p.r.ClosePath() }

func (p pen) quad(cx, cy, x, y float32) { p.r.QuadTo(cx*unit, cy*unit, x*unit, y*unit) }

func (p pen) cube(c1x, c1y, c2x, c2y, x, y float32) {
	p.r.CubeTo(c1x*unit, c1y*unit, c2x*unit, c2y*unit, x*unit, y*unit)
}

func (p pen) circle(cx, cy, r float32, hole bool) {
	k := r * kappa
	dir := float32(1)
	if hole {
		dir = -1
	}
	p.move(cx+r, cy)
	p.cube(cx+r, cy+dir*k, cx+k, cy+dir*r, cx, cy+dir*r)
	p.cube(cx-k, cy+dir*r, cx-r, cy+dir*k, cx-r, cy)
	p.cube(cx-r, cy-dir*k, cx-k, cy-dir*r, cx, cy-dir*r)
	p.cube(cx+k, cy-dir*r, cx+r, cy-dir*k, cx+r, cy)
	p.close()
}

func (p pen) roundRect(x0, y0, x1, y1, r float32) {
	k := r * (1 - kappa)
	p.move(x0+r, y0)
	p.line(x1-r, y0)
	p.cube(x1-k, y0, x1, y0+k, x1, y0+r)
	p.line(x1, y1-r)
	p.cube(x1, y1-k, x1-k, y1, x1-r, y1)
	p.line(x0+r, y1)
	p.cube(x0+k, y1, x0, y1-k, x0, y1-r)
	p.line(x0, y0+r)
	p.cube(x0, y0+k, x0+k, y0, x0+r, y0)
	p.close()
}

func fill(mask *image.Alpha, alpha float64, path func(*pen)) {
	r := vector.NewRasterizer(size, size)
	r.DrawOp = draw.Over
	path(&pen{r})
	r.Draw(mask, mask.Bounds(), image.NewUniform(color.Alpha{A: uint8(alpha*0xff + 0.5)}), image.Point{})
}

func encode(mask *image.Alpha) Glyph {
	return Glyph{Black: tinted(mask, 0), White: tinted(mask, 0xff)}
}

func tinted(mask *image.Alpha, shade uint8) []byte {
	img := image.NewNRGBA(mask.Bounds())
	for i, a := range mask.Pix {
		copy(img.Pix[i*4:], []uint8{shade, shade, shade, a})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err) // unreachable: encoding an in-memory image cannot fail
	}
	return buf.Bytes()
}
