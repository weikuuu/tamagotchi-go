// Package uifont draws Cyrillic-capable text.
//
// ebitenutil.DebugPrint's bitmap font only has ASCII glyphs, so Elysia's
// Russian lines rendered as blank boxes. This package loads a system font
// with full Unicode coverage instead and wraps ebiten's text/v2 API with a
// couple of small helpers (centered text, word-wrapped bubbles).
package uifont

import (
	"fmt"
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

// sizeScale bumps every requested font size up a bit — callers still pass
// their original logical sizes, so this is a single global "make all text
// bigger" knob instead of touching every call site.
const sizeScale = 1.2

// Unscaled converts a logical size back to what it would have rendered at
// before sizeScale existed, for the few call sites that need to opt out of
// the global bump (e.g. the stat bar labels, which should stay put).
func Unscaled(size float64) float64 {
	return size / sizeScale
}

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
	return &text.GoTextFace{Source: source, Size: size * sizeScale}
}

// boldOffsets are extra (dx, dy) passes drawn on top of the base glyph to
// fake a bolder weight, since the system fonts we load don't ship a bold
// variant of their own.
var boldOffsets = [][2]float64{{0.55, 0}, {0, 0.55}, {0.4, 0.4}}

// glyphPad is extra margin baked around a cached glyph image so the bold
// offset passes and the linear filter don't clip at the edges.
const glyphPad = 4.0

// glyphCache holds pre-rendered (string, size, color) images so repeated
// draws of the same line — a stat label, a button caption, a speech
// bubble that hasn't changed since last frame — are a cheap image blit
// instead of re-shaping the font and doing 4 draw passes every frame.
// Text rendering was the single biggest CPU cost in the app: dozens of
// labels, redrawn every frame at up to 60Hz, each one a full font-shaping
// pass. This cache turns "shape and draw" into "shape once, blit forever"
// for any line that repeats across frames (nearly all of them do).
var (
	glyphCacheMu sync.Mutex
	glyphCache   = map[string]*glyphEntry{}
)

type glyphEntry struct {
	img  *ebiten.Image
	w, h float64
}

// glyphCacheLimit is a crude cap to stop the cache from growing without
// bound over a long-running session (e.g. many distinct ambient phrases
// cycling through the speech bubble). Once hit, the whole cache is
// dropped and rebuilt lazily — simple, and cheap relative to how rarely
// it fires.
const glyphCacheLimit = 800

func glyphKey(s string, size float64, clr ebiten.ColorScale) string {
	return fmt.Sprintf("%s\x00%.2f\x00%v", s, size, clr)
}

func cachedGlyph(s string, size float64, clr ebiten.ColorScale) *glyphEntry {
	key := glyphKey(s, size, clr)

	glyphCacheMu.Lock()
	e, ok := glyphCache[key]
	glyphCacheMu.Unlock()
	if ok {
		return e
	}

	face := Face(size)
	if face == nil {
		return nil
	}
	w, h := text.Measure(s, face, size*1.2)
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = size
	}

	iw, ih := int(w+glyphPad*2)+1, int(h+glyphPad*2)+1
	img := ebiten.NewImage(iw, ih)

	draw := func(dx, dy float64) {
		opts := &text.DrawOptions{}
		opts.Filter = ebiten.FilterLinear
		opts.GeoM.Translate(glyphPad+dx, glyphPad+dy)
		opts.PrimaryAlign = text.AlignStart
		opts.ColorScale = clr
		text.Draw(img, s, face, opts)
	}
	draw(0, 0)
	for _, o := range boldOffsets {
		draw(o[0], o[1])
	}

	e = &glyphEntry{img: img, w: w, h: h}

	glyphCacheMu.Lock()
	if len(glyphCache) >= glyphCacheLimit {
		glyphCache = map[string]*glyphEntry{}
	}
	glyphCache[key] = e
	glyphCacheMu.Unlock()

	return e
}

// Draw draws a single line of text with its top-left corner at (x, y).
func Draw(dst *ebiten.Image, s string, size, x, y float64, clr ebiten.ColorScale) {
	blit(dst, s, size, x, y, text.AlignStart, clr)
}

// DrawCentered draws a single line of text horizontally centered on cx, with
// its top edge at y.
func DrawCentered(dst *ebiten.Image, s string, size, cx, y float64, clr ebiten.ColorScale) {
	blit(dst, s, size, cx, y, text.AlignCenter, clr)
}

func blit(dst *ebiten.Image, s string, size, x, y float64, align text.Align, clr ebiten.ColorScale) {
	e := cachedGlyph(s, size, clr)
	if e == nil {
		return
	}
	dx := x - glyphPad
	if align == text.AlignCenter {
		dx = x - e.w/2 - glyphPad
	}
	opts := &ebiten.DrawImageOptions{}
	opts.Filter = ebiten.FilterLinear
	opts.GeoM.Translate(dx, y-glyphPad)
	dst.DrawImage(e.img, opts)
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
