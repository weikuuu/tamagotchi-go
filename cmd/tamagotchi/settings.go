package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"tamagotchi/internal/spotify"
	"tamagotchi/internal/uifont"
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
	spot := "не подключён"
	if spotify.ClientID() != "" {
		spot = "подключён"
	}
	return []settingsRow{
		{"city", "Город", city},
		{"username", "Имя", uname},
		{"birthday", "День рождения (ММ-ДД)", bday},
		{"spotify", "Spotify Client ID", spot},
	}
}

const (
	settingsCardW     = 400
	settingsRowH      = 46
	settingsRowGapY   = 56 // vertical distance between row centers
	settingsTopPad    = 54
	settingsBottomPad = 56
)

func settingsCardRect() image.Rectangle {
	rows := 4
	h := settingsTopPad + rows*settingsRowGapY + settingsBottomPad
	x := (winWidth - settingsCardW) / 2
	y := (winHeight - h) / 2
	return image.Rect(x, y, x+settingsCardW, y+h)
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
			case "spotify":
				g.inputBuffer = spotify.ClientID()
			}
			return
		}
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

	closeR := settingsCloseRect(card)
	closeBg := color.RGBA{0xA0, 0x8A, 0xE8, 0xFF}
	if hover.In(closeR) {
		closeBg = lighten(closeBg)
	}
	vector.DrawFilledRect(screen, float32(closeR.Min.X), float32(closeR.Min.Y), float32(closeR.Dx()), float32(closeR.Dy()), closeBg, true)
	var white ebiten.ColorScale
	uifont.DrawCentered(screen, "Закрыть", 14, float64(closeR.Min.X+closeR.Dx()/2), float64(closeR.Min.Y+10), white)
}
