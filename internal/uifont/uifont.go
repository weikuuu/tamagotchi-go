// Package uifont draws Cyrillic-capable text.
//
// ebitenutil.DebugPrint's bitmap font only has ASCII glyphs, so Elysia's
// Russian lines rendered as blank boxes. This package loads a system font
// with full Unicode coverage instead and wraps ebiten's text/v2 API with a
// couple of small helpers (centered text, word-wrapped bubbles).
package uifont

import (
	"log"
	"os"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// candidatePaths are tried in order; the first one that exists and parses
// as a valid font is used. All are OS-provided fonts with full Cyrillic
// coverage, so nothing needs to be bundled or downloaded — covers macOS,
// Windows, and common Linux setups.
var candidatePaths = []string{
	// macOS
	"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
	"/System/Library/Fonts/SFNS.ttf",
	"/System/Library/Fonts/Supplemental/Tahoma.ttf",
	// Windows
	`C:\Windows\Fonts\segoeui.ttf`,
	`C:\Windows\Fonts\arial.ttf`,
	`C:\Windows\Fonts\tahoma.ttf`,
	// Linux (varies by distro; try the common ones)
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
	"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
}

var (
	once   sync.Once
	source *text.GoTextFaceSource
)

func loadSource() {
	for _, p := range candidatePaths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		src, err := text.NewGoTextFaceSource(f)
		f.Close()
		if err != nil {
			continue
		}
		source = src
		return
	}
	log.Printf("uifont: no system font found among %v; text will not render", candidatePaths)
}

// Face returns a text face at the given pixel size, or nil if no usable
// font was found.
func Face(size float64) *text.GoTextFace {
	once.Do(loadSource)
	if source == nil {
		return nil
	}
	return &text.GoTextFace{Source: source, Size: size}
}

// boldOffsets are extra (dx, dy) passes drawn on top of the base glyph to
// fake a bolder weight, since the system fonts we load don't ship a bold
// variant of their own.
var boldOffsets = [][2]float64{{0.55, 0}, {0, 0.55}, {0.4, 0.4}}

// Draw draws a single line of text with its top-left corner at (x, y).
func Draw(dst *ebiten.Image, s string, size, x, y float64, clr ebiten.ColorScale) {
	drawPasses(dst, s, size, x, y, text.AlignStart, clr)
}

// DrawCentered draws a single line of text horizontally centered on cx, with
// its top edge at y.
func DrawCentered(dst *ebiten.Image, s string, size, cx, y float64, clr ebiten.ColorScale) {
	drawPasses(dst, s, size, cx, y, text.AlignCenter, clr)
}

func drawPasses(dst *ebiten.Image, s string, size, x, y float64, align text.Align, clr ebiten.ColorScale) {
	face := Face(size)
	if face == nil {
		return
	}
	draw := func(dx, dy float64) {
		opts := &text.DrawOptions{}
		opts.GeoM.Translate(x+dx, y+dy)
		opts.PrimaryAlign = align
		opts.ColorScale = clr
		text.Draw(dst, s, face, opts)
	}
	draw(0, 0)
	for _, o := range boldOffsets {
		draw(o[0], o[1])
	}
}

// Measure returns the pixel width and height of s at the given size.
func Measure(s string, size float64) (float64, float64) {
	face := Face(size)
	if face == nil {
		return 0, 0
	}
	return text.Measure(s, face, size*1.2)
}

// Wrap breaks s into lines of at most maxWidth pixels at the given font
// size, breaking on spaces.
func Wrap(s string, size, maxWidth float64) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		candidate := cur + " " + w
		width, _ := Measure(candidate, size)
		if width > maxWidth {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur = candidate
	}
	lines = append(lines, cur)
	return lines
}

// White is a convenience fully-opaque white color scale.
func White() ebiten.ColorScale {
	var c ebiten.ColorScale
	return c
}
