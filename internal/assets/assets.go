// Package assets embeds the elygochi's art and maps it onto moods.
//
// The source art (fan illustrations of Elysia and a chibi sticker pack)
// wasn't drawn with these specific moods in mind, so most of it reads as
// upbeat. Moods without a matching illustration reuse the closest available
// art with a color tint (see Tint) rather than mismatching the artwork.
package assets

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"

	"elygochi/internal/gifanim"
	"elygochi/internal/pet"
)

//go:embed portraits
var portraitsFS embed.FS

//go:embed stickers
var stickersFS embed.FS

//go:embed frame
var frameFS embed.FS

//go:embed appicon.png
var appIconFS embed.FS

//go:embed easter_egg.jpg
var easterEggFS embed.FS

// Frame layout, in the coordinate space of the shell.png asset itself
// (1254x1254). ScreenRect is the pixel rectangle of the shell's screen
// cutout, where the portrait should be composited.
var ScreenRect = image.Rect(395, 495, 870, 885)

// ShellNativeW and ShellNativeH are shell.png's own pixel dimensions —
// ScreenRect and any other shell-relative coordinates are in this space.
const (
	ShellNativeW = 1254
	ShellNativeH = 1254
)

// LoadFrame decodes the elygochi shell artwork that the portrait sits
// inside.
func LoadFrame() (*ebiten.Image, error) {
	return decodePNG(frameFS, "frame/shell.png")
}

// BubbleTextRect is the rectangle within bubble.png (1428x634) that's safe
// for text: inside the oval, clear of the decorations and the tail.
var BubbleTextRect = image.Rect(270, 150, 1170, 420)

// LoadBubble decodes the overlay's speech-bubble artwork. Its tail points
// to the bottom-left, so it's meant to be anchored just to the right of the
// desktop pet.
func LoadBubble() (*ebiten.Image, error) {
	return decodePNG(frameFS, "frame/bubble.png")
}

// Tint is a multiplicative color meant to be applied via
// ebiten.DrawImageOptions.ColorScale, used to reshade mood-neutral art for
// moods that have no dedicated illustration.
type Tint struct{ R, G, B, A float32 }

var neutralTint = Tint{1, 1, 1, 1}

// Scale applies the tint to opts.
func (t Tint) Scale(opts *ebiten.DrawImageOptions) {
	opts.ColorScale.Scale(t.R, t.G, t.B, t.A)
}

// PortraitSet is the art shown in the main window for a given mood.
type PortraitSet struct {
	Images []*ebiten.Image
	Tint   Tint
}

// StickerSet is the animated art shown by the desktop overlay for a given
// mood.
type StickerSet struct {
	Anims []*gifanim.Animation
	Tint  Tint
}

var portraitMoods = map[pet.Mood]struct {
	files []string
	tint  Tint
}{
	pet.MoodHappy: {
		files: []string{
			"elysia.03_by_mirashi.png",
			"elysia.04_by_mirashi.png",
			"elysia.05_by_mirashi.png",
			"tumblr_2ed2618b01b72e00a75471e186358a7f_e0962921_1280.png",
			"tumblr_2a9ad283f3283fe612cf1b258ce0b336_369e582b_1280.png",
		},
		tint: neutralTint,
	},
	pet.MoodContent: {
		files: []string{
			"elysia.01_by_mirashi.png",
			"elysia.02_by_mirashi.png",
			"tumblr_0a237d9581ca7c33c56d1ea40e78c4f6_903ca064_1280.png",
			"Elysia_render10_Hoyo-Transparents.png",
		},
		tint: neutralTint,
	},
	pet.MoodBored: {
		files: []string{
			"elysia.06_by_mirashi.png",
		},
		tint: Tint{0.85, 0.85, 0.9, 1},
	},
	pet.MoodSad: {
		files: []string{
			"tumblr_923db201273f22bbfc522c3f5dd9584e_2df56a44_1280.png",
		},
		tint: Tint{0.6, 0.65, 0.9, 1},
	},
	pet.MoodHungry: {
		files: []string{
			"Elysia_render4_Hoyo-Transparents.png",
		},
		tint: neutralTint,
	},
	pet.MoodTired: {
		files: []string{
			"tumblr_70341fbdfd6aef997d0e4c397a5aebdb_dacc5a8c_540.png",
		},
		tint: Tint{0.55, 0.55, 0.65, 1},
	},
	pet.MoodSick: {
		files: []string{
			"Elysia_render10_Hoyo-Transparents.png",
		},
		tint: Tint{0.65, 0.8, 0.65, 1},
	},
}

var stickerMoods = map[pet.Mood]struct {
	files []string
	tint  Tint
}{
	pet.MoodHappy: {
		files: []string{"person.gif", "pointer.gif", "working.gif", "handwriting.gif", "1.gif", "text.gif"},
		tint:  neutralTint,
	},
	pet.MoodContent: {
		files: []string{"1.gif", "text.gif", "person.gif", "handwriting.gif", "pointer.gif"},
		tint:  neutralTint,
	},
	pet.MoodBored: {
		files: []string{"cross.gif", "text.gif", "1.gif", "working.gif"},
		tint:  Tint{0.9, 0.9, 0.95, 1},
	},
	pet.MoodSad: {
		files: []string{"unavailable.gif", "2.gif", "cross.gif", "loc.gif"},
		tint:  Tint{0.65, 0.7, 0.9, 1},
	},
	pet.MoodHungry: {
		files: []string{"help.gif", "link.gif", "2.gif", "text.gif"},
		tint:  neutralTint,
	},
	pet.MoodTired: {
		files: []string{"loc.gif", "alternate.gif", "cross.gif", "unavailable.gif"},
		tint:  Tint{0.6, 0.6, 0.7, 1},
	},
	pet.MoodSick: {
		files: []string{"cross.gif", "unavailable.gif", "loc.gif"},
		tint:  Tint{0.65, 0.85, 0.65, 1},
	},
}

// LoadPortraits decodes the main window's mood art.
func LoadPortraits() (map[pet.Mood]PortraitSet, error) {
	out := make(map[pet.Mood]PortraitSet, len(portraitMoods))
	for mood, spec := range portraitMoods {
		imgs := make([]*ebiten.Image, 0, len(spec.files))
		for _, name := range spec.files {
			img, err := decodePNG(portraitsFS, "portraits/"+name)
			if err != nil {
				return nil, fmt.Errorf("assets: portrait %s: %w", name, err)
			}
			imgs = append(imgs, img)
		}
		out[mood] = PortraitSet{Images: imgs, Tint: spec.tint}
	}
	return out, nil
}

// LoadStickers decodes the desktop overlay's mood art.
func LoadStickers() (map[pet.Mood]StickerSet, error) {
	out := make(map[pet.Mood]StickerSet, len(stickerMoods))
	for mood, spec := range stickerMoods {
		anims := make([]*gifanim.Animation, 0, len(spec.files))
		for _, name := range spec.files {
			data, err := stickersFS.ReadFile("stickers/" + name)
			if err != nil {
				return nil, fmt.Errorf("assets: sticker %s: %w", name, err)
			}
			var anim *gifanim.Animation
			if bytes.HasSuffix([]byte(name), []byte(".gif")) {
				anim, err = gifanim.Decode(data)
			} else {
				var img image.Image
				img, _, err = image.Decode(bytes.NewReader(data))
				if err == nil {
					anim = gifanim.Static(ebiten.NewImageFromImage(img))
				}
			}
			if err != nil {
				return nil, fmt.Errorf("assets: sticker %s: %w", name, err)
			}
			anims = append(anims, anim)
		}
		out[mood] = StickerSet{Anims: anims, Tint: spec.tint}
	}
	return out, nil
}

// LoadSleepSticker decodes the specific sticker that actually depicts
// Elysia sleeping (as opposed to the rest of the MoodTired set, which is
// mostly reused general-purpose art with a dim tint), for the desktop
// overlay to show right after "Спать" is pressed in the main window.
func LoadSleepSticker() (*gifanim.Animation, error) {
	data, err := stickersFS.ReadFile("stickers/alternate.gif")
	if err != nil {
		return nil, fmt.Errorf("assets: sleep sticker: %w", err)
	}
	return gifanim.Decode(data)
}

// LoadAppIcon decodes the app icon as a plain image.Image, for
// ebiten.SetWindowIcon (which sets the taskbar/title-bar icon on Windows
// and Linux — GLFW only picks up an exe's embedded icon resource if it's
// named exactly "GLFW_ICON", which our build tooling doesn't guarantee, so
// setting it explicitly at runtime is the reliable path).
func LoadAppIcon() (image.Image, error) {
	data, err := appIconFS.ReadFile("appicon.png")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// LoadEasterEgg decodes the birthday-field "67-67" easter egg image.
func LoadEasterEgg() (*ebiten.Image, error) {
	return decodePNG(easterEggFS, "easter_egg.jpg")
}

func decodePNG(fsys embed.FS, path string) (*ebiten.Image, error) {
	data, err := fsys.ReadFile(path)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}
