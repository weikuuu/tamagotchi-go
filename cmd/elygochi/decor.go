package main

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// inkColor is the dark plum used for text drawn directly on the pale pink
// background, where plain white would have no contrast.
func inkColor() ebiten.ColorScale {
	var c ebiten.ColorScale
	c.ScaleWithColor(color.RGBA{0x5A, 0x2A, 0x45, 0xFF})
	return c
}

// bubbleInkColor is a brighter, more vivid magenta used for text inside the
// speech bubbles, so it stands out more than the muted inkColor.
func bubbleInkColor() ebiten.ColorScale {
	var c ebiten.ColorScale
	c.ScaleWithColor(color.RGBA{0xD6, 0x14, 0x7A, 0xFF})
	return c
}

// bgAnchor is one point in the day's background gradient: the hour it's
// centered on, and the pale background color for that time.
type bgAnchor struct {
	hour float64
	clr  color.RGBA
}

// bgAnchors stay light enough that the existing dark-ink text (inkColor,
// bubbleInkColor) is always readable — this is an ambient tint across the
// day, not a real light/dark theme switch.
var bgAnchors = []bgAnchor{
	{2, color.RGBA{0xEB, 0xE8, 0xFA, 0xFF}},  // deep night: pale cool lavender
	{6, color.RGBA{0xEB, 0xE8, 0xFA, 0xFF}},  // pre-dawn: same as night
	{9, color.RGBA{0xFF, 0xF1, 0xE6, 0xFF}},  // morning: soft warm peach
	{13, color.RGBA{0xFF, 0xF3, 0xF7, 0xFF}}, // midday: the original pink
	{18, color.RGBA{0xFF, 0xE9, 0xE2, 0xFF}}, // evening: warm rosy sunset
	{21, color.RGBA{0xEB, 0xE8, 0xFA, 0xFF}}, // night: back to lavender
}

// bgColorForTime interpolates bgAnchors around the clock for a smooth
// gradient rather than hard cuts between time-of-day palettes.
func bgColorForTime(t time.Time) color.RGBA {
	h := t.Hour()
	m := t.Minute()
	hour := float64(h) + float64(m)/60

	n := len(bgAnchors)
	for i := 0; i < n; i++ {
		a := bgAnchors[i]
		b := bgAnchors[(i+1)%n]
		bHour := b.hour
		if bHour <= a.hour {
			bHour += 24
		}
		hh := hour
		if hh < a.hour {
			hh += 24
		}
		if hh >= a.hour && hh <= bHour {
			frac := (hh - a.hour) / (bHour - a.hour)
			return lerpColor(a.clr, b.clr, frac)
		}
	}
	return bgAnchors[0].clr
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(x, y uint8) uint8 {
		return uint8(float64(x) + (float64(y)-float64(x))*t)
	}
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), 0xFF}
}

type decoration struct {
	kind string // "heart" or "flower"
	x, y float32
	size float32
	clr  color.RGBA
}

// backgroundDecor is a fixed, hand-placed scatter of cute hearts and
// flowers in the margins around the shell, bars and buttons so nothing
// important gets covered.
var backgroundDecor = []decoration{
	{"heart", 34, 88, 9, color.RGBA{0xF2, 0x9C, 0xC3, 0xB0}},
	{"flower", 526, 83, 8, color.RGBA{0xE3, 0xA8, 0xD8, 0xB0}},
	{"heart", 532, 158, 7, color.RGBA{0xF6, 0xB8, 0xD3, 0xA0}},
	{"flower", 16, 178, 7, color.RGBA{0xC9, 0xA8, 0xE8, 0xA0}},
	{"heart", 14, 288, 6, color.RGBA{0xF2, 0x9C, 0xC3, 0x90}},
	{"flower", 544, 288, 6, color.RGBA{0xE3, 0xA8, 0xD8, 0x90}},
	{"heart", 16, 428, 7, color.RGBA{0xF6, 0xB8, 0xD3, 0xA0}},
	{"flower", 542, 448, 7, color.RGBA{0xC9, 0xA8, 0xE8, 0xA0}},
	{"flower", 46, 556, 10, color.RGBA{0xE3, 0xA8, 0xD8, 0xB0}},
	{"heart", 512, 556, 10, color.RGBA{0xF2, 0x9C, 0xC3, 0xB0}},
	{"heart", 50, 628, 8, color.RGBA{0xF6, 0xB8, 0xD3, 0xA0}},
	{"flower", 508, 633, 8, color.RGBA{0xC9, 0xA8, 0xE8, 0xA0}},
	{"flower", 28, 688, 8, color.RGBA{0xE3, 0xA8, 0xD8, 0xA0}},
	{"heart", 530, 693, 8, color.RGBA{0xF2, 0x9C, 0xC3, 0xA0}},
	{"flower", 28, 788, 9, color.RGBA{0xE3, 0xA8, 0xD8, 0xA0}},
	{"heart", 530, 793, 9, color.RGBA{0xF2, 0x9C, 0xC3, 0xA0}},
	{"heart", 280, 826, 7, color.RGBA{0xF6, 0xB8, 0xD3, 0x90}},
}

func drawBackgroundDecor(screen *ebiten.Image) {
	for _, d := range backgroundDecor {
		switch d.kind {
		case "heart":
			drawHeart(screen, d.x, d.y, d.size, d.clr)
		case "flower":
			drawFlower(screen, d.x, d.y, d.size, d.clr)
		}
	}
}

// drawHeart draws a simple cute heart centered at (x, y): two round lobes
// and a pointed bottom, built from circles and a triangle.
func drawHeart(screen *ebiten.Image, x, y, size float32, clr color.RGBA) {
	vector.FillCircle(screen, x-size*0.5, y, size*0.6, clr, true)
	vector.FillCircle(screen, x+size*0.5, y, size*0.6, clr, true)

	var path vector.Path
	path.MoveTo(x-size*1.05, y+size*0.15)
	path.LineTo(x+size*1.05, y+size*0.15)
	path.LineTo(x, y+size*1.7)
	path.Close()

	var cs ebiten.ColorScale
	cs.ScaleWithColor(clr)
	vector.FillPath(screen, &path, nil, &vector.DrawPathOptions{AntiAlias: true, ColorScale: cs})
}

// drawSparkle draws a small 4-pointed star, used for the dizzy-shake effect.
func drawSparkle(screen *ebiten.Image, x, y, size float32, clr color.RGBA) {
	const points = 4
	outer, inner := size, size*0.35

	var path vector.Path
	for i := 0; i < points*2; i++ {
		angle := float64(i) * math.Pi / points
		r := outer
		if i%2 == 1 {
			r = inner
		}
		px := x + float32(math.Cos(angle))*r
		py := y + float32(math.Sin(angle))*r
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
	path.Close()

	var cs ebiten.ColorScale
	cs.ScaleWithColor(clr)
	vector.FillPath(screen, &path, nil, &vector.DrawPathOptions{AntiAlias: true, ColorScale: cs})
}

// drawFlower draws a tiny five-petal flower centered at (x, y).
func drawFlower(screen *ebiten.Image, x, y, size float32, petal color.RGBA) {
	const petals = 5
	for i := 0; i < petals; i++ {
		angle := float64(i) / petals * 2 * math.Pi
		px := x + size*0.75*float32(math.Cos(angle))
		py := y + size*0.75*float32(math.Sin(angle))
		vector.FillCircle(screen, px, py, size*0.55, petal, true)
	}
	center := color.RGBA{0xFF, 0xE8, 0x9A, petal.A}
	vector.FillCircle(screen, x, y, size*0.5, center, true)
}
