package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"elygochi/internal/autostart"
	"elygochi/internal/bubblecfg"
	"elygochi/internal/fleecfg"
	"elygochi/internal/overlaycfg"
	"elygochi/internal/uifont"
)

type settingsRow struct {
	key, label, value string
}

// settingsRows lists the editable settings and their current display value,
// pulling live from each field's backing service/config.
func (g *mainGame) settingsRows() []settingsRow {
	city := g.weather.City()
	if city == "" {
		city = "не указан"
	}
	uname := g.username
	if uname == "" {
		uname = "не указано"
	}
	bday := g.birthdayMonthDay
	if bday == "" {
		bday = "не указан"
	}
	return []settingsRow{
		{"city", "Город", city},
		{"username", "Имя", uname},
		{"birthday", "День рождения (ДД-ММ)", bday},
	}
}

const (
	settingsCardW     = 400
	settingsRowH      = 46
	settingsRowGapY   = 56 // vertical distance between row centers
	settingsTopPad    = 54
	settingsBottomPad = 56

	settingsFieldRows = 3 // city, username, birthday
)

func settingsCardRect() image.Rectangle {
	rows := settingsFieldRows + 4 // + overlay-size stepper, flee toggle, bubble toggle, autostart toggle
	h := settingsTopPad + rows*settingsRowGapY + settingsBottomPad
	x := (winWidth - settingsCardW) / 2
	y := (winHeight - h) / 2
	return image.Rect(x, y, x+settingsCardW, y+h)
}

// settingsScaleRowRect is the row holding the overlay-size stepper, right
// below the text-field rows.
func settingsScaleRowRect(card image.Rectangle) image.Rectangle {
	return settingsRowRect(card, settingsFieldRows)
}

func settingsFleeRowRect(card image.Rectangle) image.Rectangle {
	return settingsRowRect(card, settingsFieldRows+1)
}

func settingsBubbleRowRect(card image.Rectangle) image.Rectangle {
	return settingsRowRect(card, settingsFieldRows+2)
}

func settingsAutostartRowRect(card image.Rectangle) image.Rectangle {
	return settingsRowRect(card, settingsFieldRows+3)
}

const settingsStepperBtnW = 32

func settingsScaleMinusRect(row image.Rectangle) image.Rectangle {
	y := row.Min.Y + 20
	return image.Rect(row.Min.X, y, row.Min.X+settingsStepperBtnW, y+24)
}

func settingsScalePlusRect(row image.Rectangle) image.Rectangle {
	y := row.Min.Y + 20
	return image.Rect(row.Max.X-settingsStepperBtnW, y, row.Max.X, y+24)
}

// settingsToggleRect is the clickable on/off pill on the right side of a
// toggle row (flee mode, bubble on/off).
func settingsToggleRect(row image.Rectangle) image.Rectangle {
	w, h := 90, 26
	y := row.Min.Y + (row.Dy()-h)/2 + 6
	return image.Rect(row.Max.X-w, y, row.Max.X, y+h)
}

func settingsRowRect(card image.Rectangle, i int) image.Rectangle {
	y := card.Min.Y + settingsTopPad + i*settingsRowGapY
	return image.Rect(card.Min.X+20, y, card.Max.X-20, y+settingsRowH)
}

func settingsCloseRect(card image.Rectangle) image.Rectangle {
	w, h := 120, 36
	x := card.Min.X + (card.Dx()-w)/2
	y := card.Max.Y - settingsBottomPad/2 - h/2
	return image.Rect(x, y, x+w, y+h)
}

func (g *mainGame) updateSettings() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.showSettings = false
		}
		return
	}

	mx, my := ebiten.CursorPosition()
	p := image.Pt(mx, my)

	card := settingsCardRect()
	if p.In(settingsCloseRect(card)) || !p.In(card) {
		g.showSettings = false
		return
	}

	rows := g.settingsRows()
	for i, row := range rows {
		if p.In(settingsRowRect(card, i)) {
			g.editField = row.key
			switch row.key {
			case "city":
				g.inputBuffer = g.weather.City()
			case "username":
				g.inputBuffer = g.username
			case "birthday":
				g.inputBuffer = g.birthdayMonthDay
			}
			return
		}
	}

	scaleRow := settingsScaleRowRect(card)
	switch {
	case p.In(settingsScaleMinusRect(scaleRow)):
		_ = overlaycfg.Save(overlaycfg.Clamp(overlaycfg.Load() - overlaycfg.Step))
	case p.In(settingsScalePlusRect(scaleRow)):
		_ = overlaycfg.Save(overlaycfg.Clamp(overlaycfg.Load() + overlaycfg.Step))
	}

	if p.In(settingsToggleRect(settingsFleeRowRect(card))) {
		_ = fleecfg.Save(!fleecfg.Load())
	}
	if p.In(settingsToggleRect(settingsBubbleRowRect(card))) {
		_ = bubblecfg.Save(!bubblecfg.Load())
	}
	if p.In(settingsToggleRect(settingsAutostartRowRect(card))) {
		_ = autostart.SetEnabled(!autostart.Enabled())
	}
}

func (g *mainGame) drawSettings(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, winWidth, winHeight, color.RGBA{0x24, 0x1B, 0x2E, 0xA0}, false)

	card := settingsCardRect()
	vector.DrawFilledRect(screen, float32(card.Min.X), float32(card.Min.Y), float32(card.Dx()), float32(card.Dy()), color.RGBA{0xFF, 0xF3, 0xF7, 0xFF}, true)
	vector.StrokeRect(screen, float32(card.Min.X), float32(card.Min.Y), float32(card.Dx()), float32(card.Dy()), 2, color.RGBA{0xD6, 0x14, 0x7A, 0xC0}, true)

	ink := inkColor()
	uifont.DrawCentered(screen, "Настройки", 18, float64(card.Min.X+card.Dx()/2), float64(card.Min.Y+16), ink)

	mx, my := ebiten.CursorPosition()
	hover := image.Pt(mx, my)

	for i, row := range g.settingsRows() {
		r := settingsRowRect(card, i)
		bg := color.RGBA{0xFF, 0xFF, 0xFF, 0xB0}
		if hover.In(r) {
			bg = color.RGBA{0xF6, 0xD9, 0xEA, 0xD0}
		}
		vector.DrawFilledRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), bg, true)
		vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, color.RGBA{0xC9, 0x7A, 0xA6, 0xB0}, true)
		uifont.Draw(screen, row.label, 12, float64(r.Min.X+10), float64(r.Min.Y+5), ink)
		uifont.Draw(screen, row.value, 14, float64(r.Min.X+10), float64(r.Min.Y+22), bubbleInkColor())
	}

	g.drawScaleRow(screen, card, hover)
	drawToggleRow(screen, settingsFleeRowRect(card), hover, "Режим погони (ПКМ по чибику)", fleecfg.Load())
	drawToggleRow(screen, settingsBubbleRowRect(card), hover, "Облачко с фразами у чибика", bubblecfg.Load())
	drawToggleRow(screen, settingsAutostartRowRect(card), hover, "Запуск при входе в систему", autostart.Enabled())

	closeR := settingsCloseRect(card)
	closeBg := color.RGBA{0xA0, 0x8A, 0xE8, 0xFF}
	if hover.In(closeR) {
		closeBg = lighten(closeBg)
	}
	vector.DrawFilledRect(screen, float32(closeR.Min.X), float32(closeR.Min.Y), float32(closeR.Dx()), float32(closeR.Dy()), closeBg, true)
	var white ebiten.ColorScale
	uifont.DrawCentered(screen, "Закрыть", 14, float64(closeR.Min.X+closeR.Dx()/2), float64(closeR.Min.Y+10), white)
}

// drawScaleRow renders the "-  120%  +" stepper that resizes the desktop
// overlay (a separate process — it picks the new size up within a second
// via overlaycfg).
func (g *mainGame) drawScaleRow(screen *ebiten.Image, card image.Rectangle, hover image.Point) {
	ink := inkColor()
	r := settingsScaleRowRect(card)
	vector.DrawFilledRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.RGBA{0xFF, 0xFF, 0xFF, 0xB0}, true)
	vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, color.RGBA{0xC9, 0x7A, 0xA6, 0xB0}, true)
	uifont.Draw(screen, "Размер летающей Элизии", 12, float64(r.Min.X+10), float64(r.Min.Y+5), ink)

	scale := overlaycfg.Load()

	drawStepperBtn := func(br image.Rectangle, label string) {
		bg := color.RGBA{0xA0, 0x8A, 0xE8, 0xFF}
		if hover.In(br) {
			bg = lighten(bg)
		}
		vector.DrawFilledRect(screen, float32(br.Min.X), float32(br.Min.Y), float32(br.Dx()), float32(br.Dy()), bg, true)
		var white ebiten.ColorScale
		uifont.DrawCentered(screen, label, 14, float64(br.Min.X+br.Dx()/2), float64(br.Min.Y+4), white)
	}
	drawStepperBtn(settingsScaleMinusRect(r), "-")
	drawStepperBtn(settingsScalePlusRect(r), "+")

	pct := fmt.Sprintf("%.0f%%", scale*100)
	uifont.DrawCentered(screen, pct, 14, float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+22), bubbleInkColor())
}

// drawToggleRow renders a labeled row with an "Вкл"/"Выкл" pill on the
// right; used for the flee-mode and bubble-visibility settings.
func drawToggleRow(screen *ebiten.Image, r image.Rectangle, hover image.Point, label string, on bool) {
	ink := inkColor()
	vector.DrawFilledRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.RGBA{0xFF, 0xFF, 0xFF, 0xB0}, true)
	vector.StrokeRect(screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, color.RGBA{0xC9, 0x7A, 0xA6, 0xB0}, true)
	uifont.Draw(screen, label, 12, float64(r.Min.X+10), float64(r.Min.Y+5), ink)

	tr := settingsToggleRect(r)
	bg := color.RGBA{0xB0, 0xB0, 0xB0, 0xFF}
	text := "Выкл"
	if on {
		bg = color.RGBA{0x8A, 0xD0, 0xC8, 0xFF}
		text = "Вкл"
	}
	if hover.In(tr) {
		bg = lighten(bg)
	}
	vector.DrawFilledRect(screen, float32(tr.Min.X), float32(tr.Min.Y), float32(tr.Dx()), float32(tr.Dy()), bg, true)
	var white ebiten.ColorScale
	uifont.DrawCentered(screen, text, 13, float64(tr.Min.X+tr.Dx()/2), float64(tr.Min.Y+5), white)
}
