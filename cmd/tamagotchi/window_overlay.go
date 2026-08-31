package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"tamagotchi/internal/assets"
	"tamagotchi/internal/bubblecfg"
	"tamagotchi/internal/fleecfg"
	"tamagotchi/internal/gifanim"
	"tamagotchi/internal/overlaycfg"
	"tamagotchi/internal/pet"
	"tamagotchi/internal/phrases"
	"tamagotchi/internal/uifont"
)

// Base dimensions at overlaycfg scale 1.0. The overlay's actual on-screen
// size is user-configurable (see settings.go / overlaycfg), so every one of
// these is scaled at runtime into the overlayGame fields below rather than
// used directly — see recomputeSizes.
const (
	baseOverlaySize = 74 // sprite window is a square this size at scale 1.0
	overlayMargin   = 8  // keep the pet fully on-screen

	// The bubble art is stretched wider/flatter than its native aspect
	// ratio (scaleX != scaleY) so there's more horizontal room for text
	// without the bubble growing tall.
	baseBubbleDispWidth  = 160
	baseBubbleDispHeight = 60
	baseBubbleOverlapX   = 32 // bubble slides over the sprite so its tail sits right next to it
	baseBubbleOverlapY   = 20 // vertical overlap so the tail sits right next to the sprite
	baseBubbleFontSize   = 12
	baseFleeThreshold    = 130 // px from sprite center that counts as "cursor got close"

	moodPollInterval  = 1500 * time.Millisecond
	scalePollInterval = 1 * time.Second // how often to check settings for a resize
	dragClickSlack    = 4               // pixels of movement still counted as a click, not a drag

	variantMinInterval = 4 * time.Second
	variantMaxInterval = 9 * time.Second

	overlayAmbientMinInterval = 10 * time.Second
	overlayAmbientMaxInterval = 22 * time.Second
	bubbleDuration            = 4 * time.Second

	overlayIdleAfter = 5 * time.Minute

	// Flee mode: right-click toggles it. While it's on, the pet darts away
	// whenever the cursor gets close instead of wandering on its own.
	fleeDashSpeedMult = 7.5                    // dash speed relative to the normal flightSpeed
	fleeDashDuration  = 110 * time.Millisecond // how long one fast burst lasts
	fleeDashPause     = 70 * time.Millisecond  // brief freeze between bursts — the "jerky" part
	fleeSayCooldown   = 3 * time.Second
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

	// scale and its derived pixel sizes; see recomputeSizes. Reloaded
	// periodically from overlaycfg so the settings panel in the main
	// window (a separate process) can resize this one live.
	scale          float64
	lastScalePoll  time.Time
	spriteSize     int
	bubbleW        int
	bubbleH        int
	bubbleOverlapX int
	bubbleOverlapY int
	bubbleFont     float64
	fleeThreshold  float64

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

	bubbleWrapSrc   string // "text|width" the cache below was computed from
	bubbleWrapLines []string

	canvasW, canvasH int
	extraTop         int // extra canvas height added above the sprite for the bubble

	fleeMode    bool // toggled by right-click; darts from the cursor instead of wandering
	lastFleeSay time.Time

	// Dash state machine for flee mode: alternates short fast bursts
	// (fleeDashing) with brief pauses, in the direction captured at the
	// start of the current burst.
	fleeDashing        bool
	fleeDashPhaseEnd   time.Time
	fleeDirX, fleeDirY float64

	bubbleEnabled bool // if false, she never shows the speech bubble
	lastPrefsPoll time.Time
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
	ebiten.SetWindowTitle("Elysia")
	ebiten.SetScreenClearedEveryFrame(true)
	ebiten.SetTPS(30) // a desktop pet doesn't need 60fps; halves CPU/GPU load

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
		lastState:       state,
		lastInteraction: now,
	}
	g.scale = overlaycfg.Load()
	g.fleeMode = fleecfg.Load()
	g.bubbleEnabled = bubblecfg.Load()
	g.recomputeSizes()
	g.baseX = float64(mw-g.spriteSize) / 2
	g.baseY = float64(mh-g.spriteSize) / 2
	g.canvasW = g.spriteSize
	g.canvasH = g.spriteSize

	ebiten.SetWindowSize(g.spriteSize, g.spriteSize)
	g.applyMood(state.Mood())
	g.pickNewTarget()
	g.scheduleAmbient()
	ebiten.SetWindowPosition(int(g.baseX), int(g.baseY))

	if err := ebiten.RunGame(g); err != nil {
		log.Printf("overlay: %v", err)
	}
}

// recomputeSizes derives all pixel sizes from g.scale. Call it whenever
// scale changes; the caller is responsible for resizing the window
// afterwards (updateCanvasSize picks this up automatically each frame).
func (g *overlayGame) recomputeSizes() {
	s := g.scale
	g.spriteSize = int(baseOverlaySize * s)
	g.bubbleW = int(baseBubbleDispWidth * s)
	g.bubbleH = int(baseBubbleDispHeight * s)
	g.bubbleOverlapX = int(baseBubbleOverlapX * s)
	g.bubbleOverlapY = int(baseBubbleOverlapY * s)
	g.bubbleFont = baseBubbleFontSize * s
	g.fleeThreshold = baseFleeThreshold * s
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
	g.targetX = overlayMargin + rand.Float64()*float64(g.monitorW-g.spriteSize-2*overlayMargin)
	g.targetY = overlayMargin + rand.Float64()*float64(g.monitorH-g.spriteSize-2*overlayMargin)
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

	if now.Sub(g.lastScalePoll) > scalePollInterval {
		g.lastScalePoll = now
		if newScale := overlaycfg.Load(); newScale != g.scale {
			g.scale = newScale
			g.recomputeSizes()
		}
	}
	if now.Sub(g.lastPrefsPoll) > scalePollInterval {
		g.lastPrefsPoll = now
		g.fleeMode = fleecfg.Load()
		g.bubbleEnabled = bubblecfg.Load()
	}

	if !g.nextVariant.IsZero() && now.After(g.nextVariant) {
		g.rotateVariant()
	}
	if !g.dragging && now.After(g.nextAmbient) {
		switch {
		case isNight(now):
			g.say(phrases.GetShort(phrases.Night))
		case now.Sub(g.lastInteraction) > overlayIdleAfter:
			g.say(phrases.GetShort(phrases.Idle))
		default:
			g.say(phrases.ForStateShort(g.lastState))
		}
		g.scheduleAmbient()
	}

	g.updateCanvasSize()
	g.handleMouse(now)

	if !g.dragging {
		dt := 1.0 / float64(ebiten.TPS())

		fleeing := g.fleeFromCursor(now, dt)
		if !fleeing {
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
		}

		// No bob while fleeing — a dash isn't a lazy float, it's a dead
		// sprint, and the bob just made her look like she was hopping.
		var bobY float64
		if !fleeing {
			amp, freq := bobFor(g.mood)
			g.bobPhase += dt * freq
			bobY = math.Sin(g.bobPhase*2*math.Pi) * amp
		}

		g.setWindowPos(g.baseX, g.baseY+bobY)
	}

	return nil
}

// fleeBounds returns the min/max the sprite's own screen position
// (g.baseX/g.baseY) may occupy — the same bounds setWindowPos clamps to.
// Keeping baseX/baseY themselves inside these bounds (not just the value
// actually handed to the OS) is what stops her position from silently
// drifting past a screen edge and getting stuck once the cursor backs off.
func (g *overlayGame) fleeBounds() (minX, maxX, minY, maxY float64) {
	return -overlayMargin, float64(g.monitorW - g.spriteSize + overlayMargin),
		-overlayMargin, float64(g.monitorH - g.spriteSize + overlayMargin)
}

// fleeFromCursor steers the pet away from the cursor when flee mode is on
// and the cursor is close, reports whether it did so (so the caller skips
// normal wandering for this frame). Movement is a series of short, fast
// dashes with brief pauses in between — a skittish flinch-and-freeze,
// rather than one smooth glide.
func (g *overlayGame) fleeFromCursor(now time.Time, dt float64) bool {
	if !g.fleeMode {
		return false
	}
	mx, my := ebiten.CursorPosition()
	ft := int(g.fleeThreshold)
	// CursorPosition is only meaningful near/over this tiny window; treat
	// anything further as "cursor position unknown", not "far away".
	if mx < -ft || mx > g.canvasW+ft || my < -ft || my > g.canvasH+ft {
		return false
	}

	scx, scy := float64(g.spriteSize)/2, float64(g.extraTop)+float64(g.spriteSize)/2
	dx, dy := scx-float64(mx), scy-float64(my)
	dist := math.Hypot(dx, dy)
	if dist >= g.fleeThreshold {
		return false
	}
	if dist < 1 {
		// Cursor is right on top of the sprite; pick an arbitrary direction.
		dx, dy, dist = 1, 0, 1
	}
	dx, dy = dx/dist, dy/dist

	minX, maxX, minY, maxY := g.fleeBounds()
	// Pinned against a wall and the dash would just push into it? Drop
	// that axis so she runs along the wall instead of freezing against it.
	if (g.baseX <= minX+1 && dx < 0) || (g.baseX >= maxX-1 && dx > 0) {
		dx = 0
	}
	if (g.baseY <= minY+1 && dy < 0) || (g.baseY >= maxY-1 && dy > 0) {
		dy = 0
	}
	if dx == 0 && dy == 0 {
		// Cornered. Pick a tangent so she still visibly scrambles.
		dx = 1
	}
	if n := math.Hypot(dx, dy); n > 0 {
		dx, dy = dx/n, dy/n
	}

	// Dash in short fast bursts with brief pauses, instead of one smooth
	// glide — reads as skittish little sprints rather than a slide.
	if now.After(g.fleeDashPhaseEnd) {
		g.fleeDashing = !g.fleeDashing
		if g.fleeDashing {
			g.fleeDirX, g.fleeDirY = dx, dy
			g.fleeDashPhaseEnd = now.Add(fleeDashDuration)
		} else {
			g.fleeDashPhaseEnd = now.Add(fleeDashPause)
		}
	}

	if g.fleeDashing {
		speed := flightSpeed(g.mood) * fleeDashSpeedMult
		g.baseX = clamp(g.baseX+g.fleeDirX*speed*dt, minX, maxX)
		g.baseY = clamp(g.baseY+g.fleeDirY*speed*dt, minY, maxY)
	}

	// Keep the wander target fresh so it resumes from somewhere sensible
	// once the cursor backs off.
	g.targetX, g.targetY = clamp(g.baseX+dx*40, minX, maxX), clamp(g.baseY+dy*40, minY, maxY)

	if now.Sub(g.lastFleeSay) > fleeSayCooldown {
		g.lastFleeSay = now
		g.say(phrases.GetShort(phrases.Flee))
	}
	return true
}

// updateCanvasSize grows the window up and to the right to fit the speech
// bubble while it's showing, and shrinks it back to just the sprite
// otherwise. The sprite's own screen position doesn't move; the extra room
// is added above and to the right of it. It also picks up sprite size
// changes from recomputeSizes (a scale change resizes the window even with
// no bubble showing, since wantW/wantH are then compared against the
// canvas's old size).
func (g *overlayGame) updateCanvasSize() {
	wantW, wantH, wantTop := g.spriteSize, g.spriteSize, 0
	if !g.dragging && g.bubbleEnabled && time.Now().Before(g.bubbleUntil) {
		wantW = g.spriteSize - g.bubbleOverlapX + g.bubbleW
		wantTop = g.bubbleH - g.bubbleOverlapY
		wantH = g.spriteSize + wantTop
	}
	if wantW == g.canvasW && wantH == g.canvasH && wantTop == g.extraTop {
		return
	}
	g.canvasW, g.canvasH, g.extraTop = wantW, wantH, wantTop
	ebiten.SetWindowSize(wantW, wantH)
}

func (g *overlayGame) handleMouse(now time.Time) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mx, my := ebiten.CursorPosition()
		if mx < g.spriteSize && my >= g.extraTop && my < g.extraTop+g.spriteSize {
			g.fleeMode = !g.fleeMode
			_ = fleecfg.Save(g.fleeMode)
			if g.fleeMode {
				g.say(phrases.GetShort(phrases.FleeToggleOn))
			} else {
				g.say(phrases.GetShort(phrases.FleeToggleOff))
			}
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if mx >= g.spriteSize || my < g.extraTop || my >= g.extraTop+g.spriteSize {
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
		g.say(phrases.GetShort(phrases.Streak))
	} else {
		g.say(phrases.GetShort(phrases.Petted))
	}
}

// setWindowPos positions the window given the desired screen position of
// the sprite itself (x, y); the window's actual top is extraTop pixels
// above that to make room for the speech bubble when it's showing.
func (g *overlayGame) setWindowPos(x, y float64) {
	cx := clamp(x, -overlayMargin, float64(g.monitorW-g.canvasW+overlayMargin))
	cy := clamp(y, -overlayMargin, float64(g.monitorH-g.spriteSize+overlayMargin))
	ebiten.SetWindowPosition(int(cx), int(cy)-g.extraTop)
}

func (g *overlayGame) Draw(screen *ebiten.Image) {
	if g.anim == nil {
		return
	}
	frame := g.anim.FrameAt(time.Since(g.started))
	fw, fh := frame.Bounds().Dx(), frame.Bounds().Dy()

	scale := math.Min(float64(g.spriteSize)/float64(fw), float64(g.spriteSize)/float64(fh)) * 0.9
	if time.Since(g.petFeedback) < 300*time.Millisecond {
		scale *= 1.15
	}

	opts := &ebiten.DrawImageOptions{}
	opts.Filter = ebiten.FilterNearest
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(
		(float64(g.spriteSize)-float64(fw)*scale)/2,
		float64(g.extraTop)+(float64(g.spriteSize)-float64(fh)*scale)/2,
	)
	g.tint.Scale(opts)
	screen.DrawImage(frame, opts)

	if !g.dragging && g.bubbleEnabled && time.Now().Before(g.bubbleUntil) {
		g.drawBubble(screen)
	}
}

func (g *overlayGame) drawBubble(screen *ebiten.Image) {
	bx := float64(g.spriteSize - g.bubbleOverlapX)
	by := 0.0

	bb := g.bubbleImg.Bounds()
	scaleX := float64(g.bubbleW) / float64(bb.Dx())
	scaleY := float64(g.bubbleH) / float64(bb.Dy())

	opts := &ebiten.DrawImageOptions{}
	opts.Filter = ebiten.FilterLinear
	opts.GeoM.Scale(scaleX, scaleY)
	opts.GeoM.Translate(bx, by)
	screen.DrawImage(g.bubbleImg, opts)

	tr := assets.BubbleTextRect
	textX := bx + float64(tr.Min.X)*scaleX
	textW := float64(tr.Dx()) * scaleX

	ink := bubbleInkColor()

	wrapKey := fmt.Sprintf("%s|%.1f", g.bubbleText, textW)
	if g.bubbleWrapSrc != wrapKey {
		g.bubbleWrapLines = uifont.Wrap(g.bubbleText, g.bubbleFont, textW)
		g.bubbleWrapSrc = wrapKey
	}
	lines := g.bubbleWrapLines
	lineH := g.bubbleFont + 3
	totalH := float64(len(lines)) * lineH
	textCY := by + float64(tr.Min.Y+tr.Max.Y)/2*scaleY - totalH/2
	for i, line := range lines {
		uifont.DrawCentered(screen, line, g.bubbleFont, textX+textW/2, textCY+float64(i)*lineH, ink)
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
