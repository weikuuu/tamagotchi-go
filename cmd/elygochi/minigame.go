package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"elygochi/internal/phrases"
	"elygochi/internal/uifont"
)

// A short "catch the hearts" mini-game: pixel-art hearts pop up at random
// spots for a few seconds each, click them before they vanish. Reward is
// Happiness + Affection scaled by how many you caught. Purely additive to
// the roster of actions — the regular "Играть" button still does its own
// quick thing.

const (
	miniGameDuration = 15 * time.Second
	miniHeartTTL     = 1300 * time.Millisecond
	miniHeartMinGap  = 350 * time.Millisecond
	miniHeartMaxGap  = 650 * time.Millisecond
	miniHeartBlock   = 7.0 // pixel-art block size
	miniHeartMax     = 4   // live hearts on screen at once
	miniPopDuration  = 260 * time.Millisecond
	miniResultHold   = 2500 * time.Millisecond
)

// pixelHeartPattern is a small blocky heart sprite, drawn one filled square
// per 'X'. Matches the classic 2D pixel-art heart shape.
var pixelHeartPattern = []string{
	".XX.XX.",
	"XXXXXXX",
	"XXXXXXX",
	".XXXXX.",
	"..XXX..",
	"...X...",
}

var pixelHeartW = len(pixelHeartPattern[0])
var pixelHeartH = len(pixelHeartPattern)

type miniHeart struct {
	x, y      float64
	spawnedAt time.Time
	poppedAt  time.Time // zero while alive
}

func (h miniHeart) popping() bool { return !h.poppedAt.IsZero() }

type miniGameState struct {
	endsAt     time.Time
	nextSpawn  time.Time
	hearts     []miniHeart
	score      int
	finished   bool
	finishedAt time.Time
}

// miniGameAreaRect is the play field hearts spawn inside, roughly centered
// over the shell art.
func miniGameAreaRect() image.Rectangle {
	const w, h = 460, 330
	x := (winWidth - w) / 2
	y := (winHeight - h) / 2
	return image.Rect(x, y, x+w, y+h)
}

func miniGameCloseRect() image.Rectangle {
	area := miniGameAreaRect()
	return image.Rect(area.Max.X-34, area.Min.Y-6, area.Max.X+6, area.Min.Y+34)
}

// miniHeartBounds is the clickable/visible bounding box of a heart at its
// resting (non-popping) size, used for both drawing and hit-testing so the
// two always agree.
func miniHeartBounds(cx, cy float64) image.Rectangle {
	w := float64(pixelHeartW) * miniHeartBlock
	h := float64(pixelHeartH) * miniHeartBlock
	return image.Rect(
		int(cx-w/2), int(cy-h/2),
		int(cx+w/2), int(cy+h/2),
	)
}

func (g *mainGame) startMiniGame() {
	now := time.Now()
	g.miniGame = &miniGameState{
		endsAt:    now.Add(miniGameDuration),
		nextSpawn: now,
	}
}

func (g *mainGame) updateMiniGame() {
	mg := g.miniGame
	now := time.Now()

	if mg.finished {
		if now.Sub(mg.finishedAt) > miniResultHold || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.miniGame = nil
		}
		return
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.finishMiniGame()
		return
	}
	if now.After(mg.endsAt) {
		g.finishMiniGame()
		return
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		p := image.Pt(mx, my)
		if p.In(miniGameCloseRect()) {
			g.finishMiniGame()
			return
		}
		for i := range mg.hearts {
			h := &mg.hearts[i]
			if h.popping() {
				continue
			}
			if p.In(miniHeartBounds(h.x, h.y)) {
				h.poppedAt = now
				mg.score++
				break
			}
		}
	}

	live := mg.hearts[:0]
	for _, h := range mg.hearts {
		if h.popping() {
			if now.Sub(h.poppedAt) <= miniPopDuration {
				live = append(live, h)
			}
			continue
		}
		if now.Sub(h.spawnedAt) > miniHeartTTL {
			continue
		}
		live = append(live, h)
	}
	mg.hearts = live

	area := miniGameAreaRect()
	if now.After(mg.nextSpawn) && len(mg.hearts) < miniHeartMax {
		pad := float64(pixelHeartW) * miniHeartBlock
		mg.hearts = append(mg.hearts, miniHeart{
			x:         float64(area.Min.X) + pad + rand.Float64()*(float64(area.Dx())-2*pad),
			y:         float64(area.Min.Y) + pad + rand.Float64()*(float64(area.Dy())-2*pad),
			spawnedAt: now,
		})
		mg.nextSpawn = now.Add(miniHeartMinGap + time.Duration(rand.Int63n(int64(miniHeartMaxGap-miniHeartMinGap))))
	}
}

func (g *mainGame) finishMiniGame() {
	mg := g.miniGame
	if mg.finished {
		return
	}
	mg.finished = true
	mg.finishedAt = time.Now()
	mg.hearts = nil

	g.state.PlayMiniGame(mg.score)
	g.lastInteraction = mg.finishedAt
	if streakUp := g.state.RecordActivity(mg.finishedAt); streakUp {
		g.say(phrases.Get(phrases.Streak))
	} else if mg.score >= 6 {
		g.say(phrases.Get(phrases.MiniGameGood))
	} else {
		g.say(phrases.Get(phrases.MiniGameOk))
	}
	g.store.save(g.state)
}

func (g *mainGame) drawMiniGame(screen *ebiten.Image) {
	mg := g.miniGame
	area := miniGameAreaRect()
	ink := inkColor()

	vector.DrawFilledRect(screen, 0, 0, winWidth, winHeight, color.RGBA{0x24, 0x1B, 0x2E, 0xA0}, false)
	vector.DrawFilledRect(screen, float32(area.Min.X), float32(area.Min.Y), float32(area.Dx()), float32(area.Dy()), color.RGBA{0xFF, 0xF3, 0xF7, 0xFF}, true)
	vector.StrokeRect(screen, float32(area.Min.X), float32(area.Min.Y), float32(area.Dx()), float32(area.Dy()), 2, color.RGBA{0xD6, 0x14, 0x7A, 0xC0}, true)

	if mg.finished {
		uifont.DrawCentered(screen, "Результат", 18, float64(area.Min.X+area.Dx()/2), float64(area.Min.Y+40), ink)
		uifont.DrawCentered(screen, fmt.Sprintf("Поймано сердечек: %d", mg.score), 16, float64(area.Min.X+area.Dx()/2), float64(area.Min.Y+80), bubbleInkColor())
		uifont.DrawCentered(screen, "Клик, чтобы закрыть", 12, float64(area.Min.X+area.Dx()/2), float64(area.Min.Y+area.Dy()-30), ink)
		return
	}

	uifont.DrawCentered(screen, "Лови сердечки!", 16, float64(area.Min.X+area.Dx()/2), float64(area.Min.Y+10), ink)
	uifont.DrawCentered(screen, fmt.Sprintf("Счёт: %d", mg.score), 13, float64(area.Min.X+area.Dx()/2), float64(area.Min.Y+34), bubbleInkColor())

	remaining := time.Until(mg.endsAt)
	if remaining < 0 {
		remaining = 0
	}
	frac := remaining.Seconds() / miniGameDuration.Seconds()
	barW := float32(area.Dx() - 40)
	barX := float32(area.Min.X + 20)
	barY := float32(area.Min.Y + 56)
	vector.DrawFilledRect(screen, barX, barY, barW, 8, color.RGBA{0xFF, 0xFF, 0xFF, 0xB0}, true)
	vector.DrawFilledRect(screen, barX, barY, barW*float32(frac), 8, color.RGBA{0xE8, 0x8A, 0xB4, 0xFF}, true)

	now := time.Now()
	for _, h := range mg.hearts {
		block := miniHeartBlock
		alpha := uint8(0xFF)
		if h.popping() {
			t := float64(now.Sub(h.poppedAt)) / float64(miniPopDuration)
			if t > 1 {
				t = 1
			}
			block = miniHeartBlock * (1 + t*0.9)
			alpha = uint8(0xFF * (1 - t))
		}
		fill := color.RGBA{0xE8, 0x4A, 0x8A, alpha}
		outline := color.RGBA{0xFF, 0xFF, 0xFF, alpha}
		drawPixelHeart(screen, h.x, h.y, block, fill, outline)
	}

	closeR := miniGameCloseRect()
	mx, my := ebiten.CursorPosition()
	closeBg := color.RGBA{0xA0, 0x8A, 0xE8, 0xFF}
	if image.Pt(mx, my).In(closeR) {
		closeBg = lighten(closeBg)
	}
	vector.DrawFilledRect(screen, float32(closeR.Min.X), float32(closeR.Min.Y), float32(closeR.Dx()), float32(closeR.Dy()), closeBg, true)
	var white ebiten.ColorScale
	uifont.DrawCentered(screen, "X", 14, float64(closeR.Min.X+closeR.Dx()/2), float64(closeR.Min.Y+9), white)
}

// drawPixelHeart draws pixelHeartPattern centered at (cx, cy) at the given
// block size, with a 1px outline pass underneath so it stands out against
// any background.
func drawPixelHeart(screen *ebiten.Image, cx, cy, block float64, fill, outline color.RGBA) {
	w := float64(pixelHeartW) * block
	h := float64(pixelHeartH) * block
	x0 := cx - w/2
	y0 := cy - h/2

	const pad = 1.5
	for pass := 0; pass < 2; pass++ {
		c := outline
		grow := pad
		if pass == 1 {
			c = fill
			grow = 0
		}
		for row, line := range pixelHeartPattern {
			for col, ch := range line {
				if ch != 'X' {
					continue
				}
				bx := x0 + float64(col)*block - grow
				by := y0 + float64(row)*block - grow
				vector.DrawFilledRect(screen, float32(bx), float32(by), float32(block+2*grow), float32(block+2*grow), c, false)
			}
		}
	}
}
