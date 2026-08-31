// Package gifanim decodes animated GIFs into ebiten-ready frame sequences.
package gifanim

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/gif"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// defaultFrameDelay is used when a GIF frame declares a zero delay, which
// some encoders emit to mean "as fast as possible".
const defaultFrameDelay = 100 * time.Millisecond

// minFrameDelay clamps suspiciously tiny declared delays (some sticker
// packs are authored with a handful of milliseconds per frame, relying on
// browsers to clamp them the same way) so playback doesn't flicker by way
// faster than it's meant to.
const minFrameDelay = 40 * time.Millisecond

// Animation is a decoded, ebiten-ready animated GIF.
type Animation struct {
	frames []*ebiten.Image
	delays []time.Duration
	total  time.Duration
}

// Decode parses GIF-encoded data into an Animation, compositing frames
// according to their disposal method so partial-frame GIFs render correctly.
func Decode(data []byte) (*Animation, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if len(g.Image) == 0 {
		return nil, errors.New("gifanim: no frames in GIF")
	}

	bounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	canvas := image.NewRGBA(bounds)

	anim := &Animation{}
	for i, frame := range g.Image {
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		snapshot := image.NewRGBA(bounds)
		draw.Draw(snapshot, bounds, canvas, image.Point{}, draw.Src)
		anim.frames = append(anim.frames, ebiten.NewImageFromImage(snapshot))

		d := defaultFrameDelay
		if i < len(g.Delay) && g.Delay[i] > 0 {
			d = time.Duration(g.Delay[i]) * 10 * time.Millisecond
			if d < minFrameDelay {
				d = minFrameDelay
			}
		}
		anim.delays = append(anim.delays, d)
		anim.total += d

		if i < len(g.Disposal) && g.Disposal[i] == gif.DisposalBackground {
			draw.Draw(canvas, frame.Bounds(), image.Transparent, image.Point{}, draw.Src)
		}
	}
	return anim, nil
}

// FrameAt returns the frame that should be visible after elapsed has passed
// since the animation started, looping continuously.
func (a *Animation) FrameAt(elapsed time.Duration) *ebiten.Image {
	if a.total <= 0 {
		return a.frames[0]
	}
	t := elapsed % a.total
	for i, d := range a.delays {
		if t < d {
			return a.frames[i]
		}
		t -= d
	}
	return a.frames[len(a.frames)-1]
}

// Static wraps a single image as a non-animated Animation, so callers can
// treat still images and GIFs uniformly.
func Static(img *ebiten.Image) *Animation {
	return &Animation{
		frames: []*ebiten.Image{img},
		delays: []time.Duration{time.Second},
		total:  time.Second,
	}
}

// Size returns the pixel dimensions of the animation's frames.
func (a *Animation) Size() (int, int) {
	b := a.frames[0].Bounds()
	return b.Dx(), b.Dy()
}
