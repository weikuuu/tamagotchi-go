package main

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"tamagotchi/internal/assets"
	"tamagotchi/internal/gifanim"
	"tamagotchi/internal/pet"
	"tamagotchi/internal/phrases"
	"tamagotchi/internal/uifont"
)

const (
	overlaySize   = 74 // sprite window is a fixed square this size
	overlayMargin = 8  // keep the pet fully on-screen

	// The bubble art is stretched wider/flatter than its native aspect
	// ratio (scaleX != scaleY) so there's more horizontal room for text
	// without the bubble growing tall.
	bubbleDispWidth  = 160
	bubbleDispHeight = 60
	bubbleOverlapX   = 32                                // bubble slides over the sprite so its tail sits right next to it
	bubbleOverlapY   = 20                                // vertical overlap so the tail sits right next to the sprite
	bubbleExtraTop   = bubbleDispHeight - bubbleOverlapY // extra window height added above the sprite
	bubbleFontSize   = 12

	moodPollInterval = 1500 * time.Millisecond
	dragClickSlack   = 4 // pixels of movement still counted as a click, not a drag

	variantMinInterval = 4 * time.Second
	variantMaxInterval = 9 * time.Second

	overlayAmbientMinInterval = 10 * time.Second
	overlayAmbientMaxInterval = 22 * time.Second
	bubbleDuration            = 4 * time.Second

	overlayIdleAfter = 5 * time.Minute
)

type overlayGame struct {
	store   *sharedState
	started time.Time

	stickers    map[pet.Mood]assets.StickerSet
	mood        pet.Mood
	variant     int
	anim        *gifanim.Animation
	tint        assets.Tint
	nextVariant time.Time
	lastPoll    time.Time
	lastState   pet.State

	bubbleImg *ebiten.Image

	monitorW, monitorH int

	baseX, baseY float64 // steering position, before bob offset
	targetX      float64
	targetY      float64
	pauseUntil   time.Time
	bobPhase     float64

	dragging    bool
	dragged     bool
	dragAnchorX int
	dragAnchorY int
	petFeedback time.Time

	bubbleText      string
	bubbleUntil     time.Time
	nextAmbient     time.Time
	lastInteraction time.Time

	canvasW, canvasH int
	extraTop         int // extra canvas height added above the sprite for the bubble
}

func runOverlay() {
	stickers, err := assets.LoadStickers()
	if err != nil {
		log.Fatalf("overlay: failed to load stickers: %v", err)
	}
	bubbleImg, err := assets.LoadBubble()
	if err != nil {
		log.Fatalf("overlay: failed to load bubble: %v", err)
	}

	store, state := openSharedState()

	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	ebiten.SetScreenTransparent(true)
	ebiten.SetWindowResizable(false)
	ebiten.SetWindowSize(overlaySize, overlaySize)
	ebiten.SetWindowTitle("Elysia")
	ebiten.SetScreenClearedEveryFrame(true)

	mw, mh := ebiten.Monitor().Size()
	if mw == 0 || mh == 0 {
		mw, mh = 1280, 800
	}

	now := time.Now()
	g := &overlayGame{
		store:           store,
		started:         now,
		stickers:        stickers,
		bubbleImg:       bubbleImg,
		monitorW:        mw,
		monitorH:        mh,
		baseX:           float64(mw-overlaySize) / 2,
		baseY:           float64(mh-overlaySize) / 2,
		lastState:       state,
		lastInteraction: now,
		canvasW:         overlaySize,
		canvasH:         overlaySize,
	}
	g.applyMood(state.Mood())
	g.pickNewTarget()
	g.scheduleAmbient()
	ebiten.SetWindowPosition(int(g.baseX), int(g.baseY))

	if err := ebiten.RunGame(g); err != nil {
		log.Printf("overlay: %v", err)
	}
}

func (g *overlayGame) applyMood(m pet.Mood) {
	if m == g.mood && g.anim != nil {
		return
	}
	set, ok := g.stickers[m]
	if !ok || len(set.Anims) == 0 {
		return
	}
	g.mood = m
	g.variant = rand.Intn(len(set.Anims))
	g.anim = set.Anims[g.variant]
	g.tint = set.Tint
	g.scheduleVariantChange()
}

// rotateVariant occasionally swaps to a different GIF within the current
// mood's set, so a mood with several stickers doesn't just show one of them
// forever.
func (g *overlayGame) rotateVariant() {
	set, ok := g.stickers[g.mood]
	if !ok || len(set.Anims) < 2 {
		g.scheduleVariantChange()
		return
	}
	next := rand.Intn(len(set.Anims) - 1)
	if next >= g.variant {
		next++
	}
	g.variant = next
	g.anim = set.Anims[next]
	g.scheduleVariantChange()
}

func (g *overlayGame) scheduleVariantChange() {
	g.nextVariant = time.Now().Add(variantMinInterval + time.Duration(rand.Int63n(int64(variantMaxInterval-variantMinInterval))))
}

func (g *overlayGame) scheduleAmbient() {
	g.nextAmbient = time.Now().Add(overlayAmbientMinInterval + time.Duration(rand.Int63n(int64(overlayAmbientMaxInterval-overlayAmbientMinInterval))))
}

func (g *overlayGame) say(text string) {
	g.bubbleText = text
	g.bubbleUntil = time.Now().Add(bubbleDuration)
}

func (g *overlayGame) pickNewTarget() {
	g.targetX = overlayMargin + rand.Float64()*float64(g.monitorW-overlaySize-2*overlayMargin)
	g.targetY = overlayMargin + rand.Float64()*float64(g.monitorH-overlaySize-2*overlayMargin)
}

func flightSpeed(m pet.Mood) float64 {
	switch m {
	case pet.MoodHappy:
		return 90
	case pet.MoodContent:
		return 60
	case pet.MoodBored:
		return 35
	case pet.MoodHungry:
		return 45
	default: // sad, tired, sick
		return 20
	}
}

func bobFor(m pet.Mood) (amp, freq float64) {
	switch m {
	case pet.MoodHappy:
		return 10, 3.2
	case pet.MoodContent:
		return 6, 2.2
	case pet.MoodBored:
		return 4, 1.2
	case pet.MoodHungry:
		return 5, 2.6
	default: // sad, tired, sick
		return 2, 0.8
	}
}

func (g *overlayGame) Update() error {
	now := time.Now()

	if now.Sub(g.lastPoll) > moodPollInterval {
		g.lastPoll = now
		state := g.store.load()
		state.Tick(now)
		g.lastState = state
		g.applyMood(state.Mood())
	}

	if !g.nextVariant.IsZero() && now.After(g.nextVariant) {
		g.rotateVariant()
	}
	if !g.dragging && now.After(g.nextAmbient) {
		switch {
		case isNight(now):
			g.say(phrases.Get(phrases.Night))
		case now.Sub(g.lastInteraction) > overlayIdleAfter:
			g.say(phrases.Get(phrases.Idle))
		default:
			g.say(phrases.ForState(g.lastState))
		}
		g.scheduleAmbient()
	}

	g.updateCanvasSize()
	g.handleMouse(now)

	if !g.dragging {
		dt := 1.0 / float64(ebiten.TPS())
		speed := flightSpeed(g.mood)

		dx, dy := g.targetX-g.baseX, g.targetY-g.baseY
		dist := math.Hypot(dx, dy)
		if dist < 2 {
			if g.pauseUntil.IsZero() {
				g.pauseUntil = now.Add(time.Duration(500+rand.Intn(1500)) * time.Millisecond)
			} else if now.After(g.pauseUntil) {
				g.pickNewTarget()
				g.pauseUntil = time.Time{}
			}
		} else {
			g.baseX += dx / dist * speed * dt
			g.baseY += dy / dist * speed * dt
		}

		amp, freq := bobFor(g.mood)
		g.bobPhase += dt * freq
		bobY := math.Sin(g.bobPhase*2*math.Pi) * amp

		g.setWindowPos(g.baseX, g.baseY+bobY)
	}

	return nil
}

// updateCanvasSize grows the window up and to the right to fit the speech
// bubble while it's showing, and shrinks it back to just the sprite
// otherwise. The sprite's own screen position doesn't move; the extra room
// is added above and to the right of it.
func (g *overlayGame) updateCanvasSize() {
	wantW, wantH, wantTop := overlaySize, overlaySize, 0
	if !g.dragging && time.Now().Before(g.bubbleUntil) {
		wantW = overlaySize - bubbleOverlapX + bubbleDispWidth
		wantTop = bubbleExtraTop
		wantH = overlaySize + wantTop
	}
	if wantW == g.canvasW && wantH == g.canvasH && wantTop == g.extraTop {
		return
	}
	g.canvasW, g.canvasH, g.extraTop = wantW, wantH, wantTop
	ebiten.SetWindowSize(wantW, wantH)
}

func (g *overlayGame) handleMouse(now time.Time) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if mx >= overlaySize || my < g.extraTop || my >= g.extraTop+overlaySize {
			// Click landed on the speech bubble, not the pet.
			return
		}
		g.dragging = true
		g.dragged = false
		g.dragAnchorX, g.dragAnchorY = mx, my
	}
	if g.dragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		lx, ly := ebiten.CursorPosition()
		dx, dy := lx-g.dragAnchorX, ly-g.dragAnchorY
		if abs(dx) > dragClickSlack || abs(dy) > dragClickSlack {
			g.dragged = true
		}
		wx, wy := ebiten.WindowPosition()
		nx, ny := wx+dx, wy+dy
		ebiten.SetWindowPosition(nx, ny)
		g.baseX, g.baseY = float64(nx), float64(ny)
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if g.dragging && !g.dragged {
			g.petPet(now)
		} else if g.dragging {
			// Resume autonomous flight from wherever it was dropped.
			g.pickNewTarget()
			g.pauseUntil = time.Time{}
		}
		g.dragging = false
	}
}

func (g *overlayGame) petPet(now time.Time) {
	state := g.store.load()
	state.Tick(now)
	state.Pet()
	streakUp := state.RecordActivity(now)
	g.store.save(state)
	g.lastState = state
	g.lastInteraction = now
	g.applyMood(state.Mood())
	g.petFeedback = now
	if streakUp {
		g.say(phrases.Get(phrases.Streak))
	} else {
		g.say(phrases.Get(phrases.Petted))
	}
}

// setWindowPos positions the window given the desired screen position of
// the sprite itself (x, y); the window's actual top is extraTop pixels
// above that to make room for the speech bubble when it's showing.
func (g *overlayGame) setWindowPos(x, y float64) {
	cx := clamp(x, -overlayMargin, float64(g.monitorW-g.canvasW+overlayMargin))
	cy := clamp(y, -overlayMargin, float64(g.monitorH-overlaySize+overlayMargin))
	ebiten.SetWindowPosition(int(cx), int(cy)-g.extraTop)
}

func (g *overlayGame) Draw(screen *ebiten.Image) {
	if g.anim == nil {
		return
	}
	frame := g.anim.FrameAt(time.Since(g.started))
	fw, fh := frame.Bounds().Dx(), frame.Bounds().Dy()

	scale := math.Min(float64(overlaySize)/float64(fw), float64(overlaySize)/float64(fh)) * 0.9
	if time.Since(g.petFeedback) < 300*time.Millisecond {
		scale *= 1.15
	}

	opts := &ebiten.DrawImageOptions{}
	opts.Filter = ebiten.FilterNearest
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(
		(float64(overlaySize)-float64(fw)*scale)/2,
		float64(g.extraTop)+(float64(overlaySize)-float64(fh)*scale)/2,
	)
	g.tint.Scale(opts)
	screen.DrawImage(frame, opts)

	if !g.dragging && time.Now().Before(g.bubbleUntil) {
		g.drawBubble(screen)
	}
}

func (g *overlayGame) drawBubble(screen *ebiten.Image) {
	bx := float64(overlaySize - bubbleOverlapX)
	by := 0.0

	bb := g.bubbleImg.Bounds()
	scaleX := float64(bubbleDispWidth) / float64(bb.Dx())
	scaleY := float64(bubbleDispHeight) / float64(bb.Dy())

	opts := &ebiten.DrawImageOptions{}
	opts.Filter = ebiten.FilterLinear
	opts.GeoM.Scale(scaleX, scaleY)
	opts.GeoM.Translate(bx, by)
	screen.DrawImage(g.bubbleImg, opts)

	tr := assets.BubbleTextRect
	textX := bx + float64(tr.Min.X)*scaleX
	textW := float64(tr.Dx()) * scaleX

	ink := bubbleInkColor()

	lines := uifont.Wrap(g.bubbleText, bubbleFontSize, textW)
	lineH := float64(bubbleFontSize + 3)
	totalH := float64(len(lines)) * lineH
	textCY := by + float64(tr.Min.Y+tr.Max.Y)/2*scaleY - totalH/2
	for i, line := range lines {
		uifont.DrawCentered(screen, line, bubbleFontSize, textX+textW/2, textCY+float64(i)*lineH, ink)
	}
}

func (g *overlayGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.canvasW, g.canvasH
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
