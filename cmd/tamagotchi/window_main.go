package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"tamagotchi/internal/assets"
	"tamagotchi/internal/birthday"
	"tamagotchi/internal/pet"
	"tamagotchi/internal/phrases"
	"tamagotchi/internal/spotify"
	"tamagotchi/internal/sysinfo"
	"tamagotchi/internal/uifont"
	"tamagotchi/internal/username"
	"tamagotchi/internal/weather"
)

const (
	winWidth  = 560
	winHeight = 842

	infoBarHeight = 26 // top strip showing clock/weather/activity; click to edit the city
	nameZoneWidth = 96 // right-hand slice of the info bar; click to edit your name

	nowPlayingY      = infoBarHeight + 4
	nowPlayingHeight = 24 // dedicated row for "what's playing", separate from the crowded info bar

	shellWidth = 440 // the tamagotchi shell art is scaled to this width
	shellX     = (winWidth - shellWidth) / 2
	shellY     = nowPlayingY + nowPlayingHeight + 12

	barWidth  = 340
	barHeight = 18
	barGap    = 34

	buttonWidth  = 104
	buttonHeight = 48
	buttonGap    = 12

	messageDuration    = 4 * time.Second
	ambientMinInterval = 12 * time.Second
	ambientMaxInterval = 25 * time.Second

	sleepAnimDuration = 4 * time.Second // played right after pressing "Спать"

	idleThreshold     = 3 * time.Minute
	longAbsenceMinGap = 6 * time.Hour

	settingsButtonAreaHeight = 56 // bottom margin below the buttons, holds the "Настройки" link
)

// shellHeight is derived from the source art's aspect ratio (1040x1112).
var shellHeight = func() int {
	w := float64(shellWidth)
	return int(w * 1112 / 1040)
}()

type button struct {
	label     string
	actionKey string
	rect      image.Rectangle
	action    func(*pet.State)
	color     color.RGBA
}

type mainGame struct {
	store    *sharedState
	state    pet.State
	lastSave time.Time

	portraits map[pet.Mood]assets.PortraitSet
	frame     *ebiten.Image
	bubbleImg *ebiten.Image
	screenBox image.Rectangle // scaled screen cutout, in window coordinates
	buttons   []button

	weather *weather.Service
	sysinfo *sysinfo.Service
	spotify *spotify.Service

	editField    string // "" | "city" | "username" | "birthday" | "spotify" — which text field is being edited
	inputBuffer  string
	showSettings bool

	lastTrackKey string // "track|artist" last seen, to detect changes

	message         string
	messageStart    time.Time
	messageUntil    time.Time
	nextAmbient     time.Time
	lastInteraction time.Time

	sleepUntil time.Time // active right after pressing "Спать"

	birthdayMonthDay string // "MM-DD", loaded once at startup; "" if unset
	username         string // loaded once at startup; "" if unset
}

func runMainWindow() {
	portraits, err := assets.LoadPortraits()
	if err != nil {
		log.Fatalf("main window: failed to load portraits: %v", err)
	}
	frame, err := assets.LoadFrame()
	if err != nil {
		log.Fatalf("main window: failed to load frame: %v", err)
	}
	bubbleImg, err := assets.LoadBubble()
	if err != nil {
		log.Fatalf("main window: failed to load bubble: %v", err)
	}

	store, state := openSharedState()
	now := time.Now()
	g := &mainGame{
		store:            store,
		state:            state,
		portraits:        portraits,
		frame:            frame,
		bubbleImg:        bubbleImg,
		lastInteraction:  now,
		weather:          weather.Start(),
		sysinfo:          sysinfo.Start(),
		spotify:          spotify.Start(),
		birthdayMonthDay: birthday.Load(),
		username:         username.Load(),
	}
	g.screenBox = scaleRect(assets.ScreenRect, shellWidth, shellHeight, shellX, shellY)

	if !state.LastTick.IsZero() && now.Sub(state.LastTick) > longAbsenceMinGap {
		g.say(phrases.Get(phrases.LongAbsence))
	}
	g.buttons = []button{
		{label: "Покормить", actionKey: "feed", action: (*pet.State).Feed, color: color.RGBA{0xE8, 0x8A, 0xB4, 0xFF}},
		{label: "Играть", actionKey: "play", action: (*pet.State).Play, color: color.RGBA{0x8A, 0xB4, 0xE8, 0xFF}},
		{label: "Спать", actionKey: "rest", action: (*pet.State).Rest, color: color.RGBA{0xA0, 0x8A, 0xE8, 0xFF}},
		{label: "Искупать", actionKey: "wash", action: (*pet.State).Wash, color: color.RGBA{0x8A, 0xD0, 0xC8, 0xFF}},
	}
	g.scheduleAmbient()
	buttonY := winHeight - buttonHeight - settingsButtonAreaHeight
	totalW := len(g.buttons)*buttonWidth + (len(g.buttons)-1)*buttonGap
	x := (winWidth - totalW) / 2
	for i := range g.buttons {
		g.buttons[i].rect = image.Rect(x, buttonY, x+buttonWidth, buttonY+buttonHeight)
		x += buttonWidth + buttonGap
	}

	ebiten.SetWindowSize(winWidth, winHeight)
	ebiten.SetWindowTitle("Elysia — tamagotchi")
	ebiten.SetWindowResizable(false)

	if err := ebiten.RunGame(g); err != nil {
		log.Printf("main window: %v", err)
	}
	g.store.save(g.state)
}

// scaleRect maps a rectangle from the shell art's native 1040x1112 space
// into window coordinates, given the displayed shell size and offset.
func scaleRect(r image.Rectangle, dispW, dispH, offX, offY int) image.Rectangle {
	const nativeW, nativeH = 1040, 1112
	sx := float64(dispW) / nativeW
	sy := float64(dispH) / nativeH
	return image.Rect(
		offX+int(float64(r.Min.X)*sx),
		offY+int(float64(r.Min.Y)*sy),
		offX+int(float64(r.Max.X)*sx),
		offY+int(float64(r.Max.Y)*sy),
	)
}

func (g *mainGame) Update() error {
	now := time.Now()
	g.state.Tick(now)

	if g.editField != "" {
		g.updateTextInput()
		return nil
	}

	if g.showSettings {
		g.updateSettings()
		return nil
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		p := image.Pt(mx, my)

		if my < infoBarHeight {
			if mx >= winWidth-nameZoneWidth {
				g.editField = "username"
				g.inputBuffer = g.username
			} else {
				g.editField = "city"
				g.inputBuffer = g.weather.City()
			}
			return nil
		}

		if p.In(settingsButtonRect()) {
			g.showSettings = true
			return nil
		}

		for _, b := range g.buttons {
			if p.In(b.rect) {
				b.action(&g.state)
				g.lastInteraction = now
				if streakUp := g.state.RecordActivity(now); streakUp {
					g.say(phrases.Get(phrases.Streak))
				} else {
					g.say(phrases.ForAction(b.actionKey, g.state))
				}
				if b.actionKey == "rest" {
					g.sleepUntil = now.Add(sleepAnimDuration)
				}
				g.store.save(g.state)
				break
			}
		}
	}

	if snap := g.spotify.Snapshot(); snap.Ready && snap.Playing {
		key := snap.Track + "|" + snap.Artist
		if g.lastTrackKey != "" && key != g.lastTrackKey {
			g.say(phrases.TrackReaction(snap.Artist))
		}
		g.lastTrackKey = key
	}

	if now.After(g.nextAmbient) {
		switch {
		case birthday.IsToday(g.birthdayMonthDay, now):
			g.say(phrases.Get(phrases.Birthday))
		case birthday.IsNewYear(now):
			g.say(phrases.Get(phrases.NewYear))
		case isNight(now):
			g.say(phrases.Get(phrases.Night))
		case now.Sub(g.lastInteraction) > idleThreshold:
			g.say(phrases.Get(phrases.Idle))
		case g.spotify.Snapshot().Playing && rand.Float64() < 0.4:
			g.say(phrases.Get(phrases.Music))
		case g.username != "" && rand.Float64() < 0.2:
			g.say(phrases.WithName(g.username))
		default:
			g.say(phrases.ForState(g.state))
		}
		g.scheduleAmbient()
	}

	if now.Sub(g.lastSave) > 5*time.Second {
		g.store.save(g.state)
		g.lastSave = now
	}
	return nil
}

func (g *mainGame) updateTextInput() {
	g.inputBuffer += string(ebiten.AppendInputChars(nil))
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.inputBuffer) > 0 {
		r := []rune(g.inputBuffer)
		g.inputBuffer = string(r[:len(r)-1])
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		if g.inputBuffer != "" {
			switch g.editField {
			case "city":
				g.weather.SetCity(g.inputBuffer)
			case "username":
				g.username = g.inputBuffer
				_ = username.Save(g.username)
			case "birthday":
				if err := birthday.Save(g.inputBuffer); err == nil {
					g.birthdayMonthDay = g.inputBuffer
				}
			case "spotify":
				_ = spotify.SaveClientID(g.inputBuffer)
				g.spotify = spotify.Start()
			}
		}
		g.editField = ""
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.editField = ""
	}
}

func (g *mainGame) say(text string) {
	g.message = text
	g.messageStart = time.Now()
	g.messageUntil = g.messageStart.Add(messageDuration)
}

func isNight(t time.Time) bool {
	h := t.Local().Hour()
	return h >= 23 || h < 6
}

func (g *mainGame) scheduleAmbient() {
	g.nextAmbient = time.Now().Add(ambientMinInterval + time.Duration(rand.Int63n(int64(ambientMaxInterval-ambientMinInterval))))
}

func (g *mainGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0xFF, 0xF3, 0xF7, 0xFF})
	drawBackgroundDecor(screen)

	var white ebiten.ColorScale
	ink := inkColor()
	if g.editField != "" {
		g.drawTextInput(screen, white)
	} else {
		uifont.DrawCentered(screen, g.infoLine(), 13, winWidth/2, 10, ink)
		nameLabel := "+ имя"
		if g.username != "" {
			nameLabel = g.username
		}
		uifont.DrawCentered(screen, nameLabel, 12, float64(winWidth-nameZoneWidth/2), 10, ink)
	}

	g.drawNowPlaying(screen)

	// Shell art.
	sopts := &ebiten.DrawImageOptions{}
	sopts.Filter = ebiten.FilterLinear
	fb := g.frame.Bounds()
	scale := float64(shellWidth) / float64(fb.Dx())
	sopts.GeoM.Scale(scale, scale)
	sopts.GeoM.Translate(shellX, shellY)
	screen.DrawImage(g.frame, sopts)

	// Portrait, composited into the shell's screen cutout.
	mood := g.state.Mood()
	now := time.Now()
	sleeping := mood == pet.MoodTired || now.Before(g.sleepUntil)
	set, ok := g.portraits[mood]
	if ok && len(set.Images) > 0 {
		variant := int(now.Unix()/6) % len(set.Images)
		img := set.Images[variant]

		var yOff float64
		if now.Before(g.messageUntil) {
			amp := bobAmplitude(mood)
			yOff = math.Sin(time.Since(g.messageStart).Seconds()*6) * amp
		} else if sleeping {
			// Slow, gentle breathing motion while asleep.
			yOff = math.Sin(float64(now.UnixMilli())/1000*1.2) * 3
		}

		boxW, boxH := float64(g.screenBox.Dx()), float64(g.screenBox.Dy())
		iw, ih := img.Bounds().Dx(), img.Bounds().Dy()
		pscale := math.Min(boxW/float64(iw), boxH/float64(ih))
		dw, dh := float64(iw)*pscale, float64(ih)*pscale

		opts := &ebiten.DrawImageOptions{}
		opts.Filter = ebiten.FilterLinear
		opts.GeoM.Scale(pscale, pscale)
		opts.GeoM.Translate(
			float64(g.screenBox.Min.X)+(boxW-dw)/2,
			float64(g.screenBox.Min.Y)+(boxH-dh)/2+yOff,
		)
		set.Tint.Scale(opts)
		if now.Before(g.sleepUntil) {
			opts.ColorScale.Scale(0.75, 0.75, 0.85, 1)
		}
		screen.DrawImage(img, opts)
	}
	if sleeping {
		drawSleepAnim(screen, g.screenBox)
	}

	barsTop := shellY + shellHeight + 20
	drawBar(screen, "Голод", float64(g.state.Hunger)/float64(pet.MaxStat), barsTop, color.RGBA{0xE8, 0x8A, 0xB4, 0xFF})
	drawBar(screen, "Энергия", float64(g.state.Energy)/float64(pet.MaxStat), barsTop+barGap, color.RGBA{0x8A, 0xB4, 0xE8, 0xFF})
	drawBar(screen, "Радость", float64(g.state.Happiness)/float64(pet.MaxStat), barsTop+2*barGap, color.RGBA{0xE8, 0xD8, 0x8A, 0xFF})
	drawBar(screen, "Чистота", float64(g.state.Cleanliness)/float64(pet.MaxStat), barsTop+3*barGap, color.RGBA{0x8A, 0xD0, 0xC8, 0xFF})

	uifont.DrawCentered(screen, moodLabel(mood), 16, winWidth/2, float64(barsTop+4*barGap+2), ink)
	if g.state.StreakDays > 0 {
		uifont.DrawCentered(screen, streakLabel(g.state.StreakDays), 13, winWidth/2, float64(barsTop+4*barGap+26), ink)
	}

	mx, my := ebiten.CursorPosition()
	hover := image.Pt(mx, my)
	for _, b := range g.buttons {
		c := b.color
		if hover.In(b.rect) {
			c = lighten(c)
		}
		vector.DrawFilledRect(screen, float32(b.rect.Min.X), float32(b.rect.Min.Y), float32(b.rect.Dx()), float32(b.rect.Dy()), c, true)
		uifont.DrawCentered(screen, b.label, 16, float64(b.rect.Min.X+b.rect.Dx()/2), float64(b.rect.Min.Y+b.rect.Dy()/2-11), white)
	}

	sr := settingsButtonRect()
	uifont.DrawCentered(screen, "⚙ Настройки", 13, float64(sr.Min.X+sr.Dx()/2), float64(sr.Min.Y+4), ink)

	if time.Now().Before(g.messageUntil) {
		g.drawBubble(screen)
	}

	if g.showSettings {
		g.drawSettings(screen)
	}
}

// settingsButtonRect is the clickable "⚙ Настройки" link in the bottom
// margin below the action buttons.
func settingsButtonRect() image.Rectangle {
	w, h := 120, 24
	y := winHeight - settingsButtonAreaHeight/2 - h/2
	return image.Rect(winWidth/2-w/2, y, winWidth/2+w/2, y+h)
}

// drawBubble shows the current line in the same cute speech-bubble art the
// desktop overlay uses, anchored just above and right of Elysia's head so
// its tail points back at her.
func (g *mainGame) drawBubble(screen *ebiten.Image) {
	const bw = 210.0
	bh := bw * 634 / 1428

	bx := float64(g.screenBox.Max.X) - 28
	by := float64(g.screenBox.Min.Y) - bh + 16
	if bx+bw > winWidth-8 {
		bx = winWidth - 8 - bw
	}
	if by < float64(infoBarHeight+4) {
		by = float64(infoBarHeight + 4)
	}

	bb := g.bubbleImg.Bounds()
	scale := bw / float64(bb.Dx())

	opts := &ebiten.DrawImageOptions{}
	opts.Filter = ebiten.FilterLinear
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(bx, by)
	screen.DrawImage(g.bubbleImg, opts)

	tr := assets.BubbleTextRect
	textX := bx + float64(tr.Min.X)*scale
	textW := float64(tr.Dx()) * scale

	ink := bubbleInkColor()
	lines := uifont.Wrap(g.message, 14, textW)
	lineH := 17.0
	totalH := float64(len(lines)) * lineH
	textCY := by + float64(tr.Min.Y+tr.Max.Y)/2*scale - totalH/2
	for i, line := range lines {
		uifont.DrawCentered(screen, line, 14, textX+textW/2, textCY+float64(i)*lineH, ink)
	}
}

func (g *mainGame) drawTextInput(screen *ebiten.Image, white ebiten.ColorScale) {
	vector.DrawFilledRect(screen, 40, 2, winWidth-80, infoBarHeight-4, color.RGBA{0x3A, 0x2C, 0x47, 0xFF}, true)
	vector.StrokeRect(screen, 40, 2, winWidth-80, infoBarHeight-4, 1, color.RGBA{0xFF, 0xFF, 0xFF, 0xA0}, true)

	placeholder := "Введите город и нажмите Enter"
	switch g.editField {
	case "username":
		placeholder = "Введите имя и нажмите Enter"
	case "birthday":
		placeholder = "Введите день рождения как ММ-ДД и нажмите Enter"
	case "spotify":
		placeholder = "Вставьте Spotify Client ID и нажмите Enter"
	}

	text := g.inputBuffer
	blink := int(time.Now().UnixMilli()/500)%2 == 0
	if blink {
		text += "|"
	}
	if g.inputBuffer == "" && !blink {
		text = placeholder
	}
	uifont.DrawCentered(screen, text, 13, winWidth/2, 9, white)
}

func (g *mainGame) infoLine() string {
	city := g.weather.City()
	parts := []string{time.Now().Local().Format("15:04")}

	switch {
	case city == "":
		parts = append(parts, "нажмите, чтобы указать город")
	case g.weather.Snapshot().Ready():
		snap := g.weather.Snapshot()
		parts = append(parts, fmt.Sprintf("%s, %.0f°C, %s", snap.City, snap.TempC, snap.Desc))
	default:
		parts = append(parts, city+"…")
	}

	if app := g.sysinfo.ActiveApp(); app != "" {
		parts = append(parts, app)
	}

	line := parts[0]
	for _, p := range parts[1:] {
		line += "  ·  " + p
	}
	return line
}

// streakLabel formats the day-streak counter with correct Russian
// pluralization for "день".
func streakLabel(days int) string {
	return fmt.Sprintf("Серия: %d %s", days, ruDays(days))
}

func ruDays(n int) string {
	n = n % 100
	if n >= 11 && n <= 14 {
		return "дней"
	}
	switch n % 10 {
	case 1:
		return "день"
	case 2, 3, 4:
		return "дня"
	default:
		return "дней"
	}
}

// drawNowPlaying shows a dedicated "what's playing on Spotify" pill below
// the info bar, so it doesn't have to fight for space with the
// clock/weather/activity line.
func (g *mainGame) drawNowPlaying(screen *ebiten.Image) {
	snap := g.spotify.Snapshot()
	if !snap.Playing {
		return
	}
	x := float32(20)
	w := float32(winWidth - 40)
	vector.DrawFilledRect(screen, x, nowPlayingY, w, nowPlayingHeight, color.RGBA{0xE3, 0xA8, 0xD8, 0x80}, true)
	vector.StrokeRect(screen, x, nowPlayingY, w, nowPlayingHeight, 1, color.RGBA{0xD6, 0x14, 0x7A, 0x90}, true)

	text := fmt.Sprintf("♪ %s — %s", snap.Track, snap.Artist)
	maxW := float64(w) - 16
	for tw, _ := uifont.Measure(text, 13); tw > maxW && len([]rune(text)) > 4; tw, _ = uifont.Measure(text, 13) {
		r := []rune(text)
		text = string(r[:len(r)-2]) + "…"
	}
	uifont.DrawCentered(screen, text, 13, winWidth/2, nowPlayingY+5, inkColor())
}

func moodLabel(m pet.Mood) string {
	names := map[pet.Mood]string{
		pet.MoodHappy:   "счастлива",
		pet.MoodContent: "довольна",
		pet.MoodBored:   "скучает",
		pet.MoodSad:     "грустит",
		pet.MoodHungry:  "голодна",
		pet.MoodTired:   "устала",
		pet.MoodSick:    "плохо себя чувствует",
	}
	if s, ok := names[m]; ok {
		return "Настроение: " + s
	}
	return fmt.Sprintf("Настроение: %s", m)
}

func (g *mainGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return winWidth, winHeight
}

// drawSleepAnim draws three "z"s drifting up and fading out in a loop from
// the top-right corner of box, in the classic "sleeping" cartoon style.
func drawSleepAnim(screen *ebiten.Image, box image.Rectangle) {
	const cycle = 2.2 // seconds per "z"
	t := float64(time.Now().UnixMilli()) / 1000

	for i := 0; i < 3; i++ {
		phase := math.Mod(t/cycle+float64(i)/3, 1)
		size := 12 + phase*14
		x := float64(box.Max.X) - 26 + float64(i)*10
		y := float64(box.Min.Y) + 24 - phase*46

		var cs ebiten.ColorScale
		cs.ScaleAlpha(float32(1 - phase))
		uifont.DrawCentered(screen, "z", size, x, y, cs)
	}
}

func bobAmplitude(m pet.Mood) float64 {
	switch m {
	case pet.MoodHappy:
		return 10
	case pet.MoodContent:
		return 6
	case pet.MoodBored:
		return 3
	default:
		return 1.5
	}
}

func drawBar(screen *ebiten.Image, label string, frac float64, y int, c color.RGBA) {
	x := (winWidth - barWidth) / 2
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(barWidth), float32(barHeight), color.RGBA{0xFF, 0xFF, 0xFF, 0xB0}, true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(barWidth), float32(barHeight), 2, color.RGBA{0xC9, 0x7A, 0xA6, 0xB0}, true)
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	vector.DrawFilledRect(screen, float32(x+2), float32(y+2), float32(frac*(barWidth-4)), float32(barHeight-4), c, true)
	uifont.Draw(screen, label, 13, float64(x), float64(y-17), inkColor())
}

func lighten(c color.RGBA) color.RGBA {
	blend := func(v uint8) uint8 {
		f := float64(v) + (255-float64(v))*0.25
		if f > 255 {
			f = 255
		}
		return uint8(f)
	}
	return color.RGBA{blend(c.R), blend(c.G), blend(c.B), c.A}
}
