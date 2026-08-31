// Command cropassets normalizes the portrait PNGs to a common 3:4 aspect
// ratio so they read as a consistent "card" no matter how the source art
// was originally framed (some are tight face shots, some full-body, some
// wide scene renders).
//
// For each image it:
//  1. finds the bounding box of non-transparent pixels (falls back to the
//     full image for opaque backgrounds),
//  2. picks the largest 3:4 rectangle that fits inside that box, centered
//     on the alpha-weighted centroid horizontally and top-aligned
//     vertically (art in this set is consistently head-at-top),
//  3. overwrites the file with the crop.
//
// Run from the repo root: go run ./internal/tools/cropassets
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

const targetAspect = 3.0 / 4.0 // width:height

// bgColorThreshold is how close a pixel's color needs to be to the sampled
// border color to be treated as background during flood fill.
const bgColorThreshold = 26.0

func main() {
	dir := "internal/assets/portraits"

	var paths []string
	if len(os.Args) > 1 {
		for _, name := range os.Args[1:] {
			paths = append(paths, filepath.Join(dir, name))
		}
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			log.Fatalf("cropassets: %v", err)
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".png" {
				continue
			}
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}

	for _, path := range paths {
		if err := cropFile(path); err != nil {
			log.Fatalf("cropassets: %s: %v", path, err)
		}
		fmt.Println("cropped", path)
	}
}

func cropFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}

	rgba := toRGBA(src)
	removeBackground(rgba)

	rect := cropRect(rgba)

	out := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(out, out.Bounds(), rgba, rect.Min, draw.Src)

	tmp := path + ".tmp"
	wf, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := png.Encode(wf, out); err != nil {
		wf.Close()
		return err
	}
	if err := wf.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func toRGBA(src image.Image) *image.RGBA {
	if r, ok := src.(*image.RGBA); ok {
		return r
	}
	b := src.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	return out
}

// removeBackground flood-fills from the image border, turning any pixel
// connected to the edge and close in color to it transparent. This clears
// flat studio backdrops (black, white, ...) without touching enclosed dark
// areas like hair or clothing, and is a no-op on pixels already
// transparent or on backgrounds that aren't actually flat.
func removeBackground(img *image.RGBA) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return
	}

	at := func(x, y int) color.RGBA {
		return img.RGBAAt(b.Min.X+x, b.Min.Y+y)
	}

	border, ok := averageOpaqueBorderColor(w, h, at)
	if !ok {
		// Border is already fully transparent; nothing to remove.
		return
	}

	visited := make([]bool, w*h)
	var stack [][2]int
	seed := func(x, y int) {
		i := y*w + x
		if visited[i] {
			return
		}
		visited[i] = true
		stack = append(stack, [2]int{x, y})
	}
	for x := 0; x < w; x++ {
		seed(x, 0)
		seed(x, h-1)
	}
	for y := 0; y < h; y++ {
		seed(0, y)
		seed(w-1, y)
	}

	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		x, y := p[0], p[1]
		c := at(x, y)
		if c.A == 0 {
			continue
		}
		if !colorClose(c, border) {
			continue
		}
		img.SetRGBA(b.Min.X+x, b.Min.Y+y, color.RGBA{})

		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			i := ny*w + nx
			if visited[i] {
				continue
			}
			visited[i] = true
			stack = append(stack, [2]int{nx, ny})
		}
	}
}

// averageOpaqueBorderColor averages the opaque pixels along the image edge
// to get a robust reference background color, tolerant of slight vignette
// or compression noise. ok is false if the edge has no opaque pixels at
// all (i.e. the image is already transparent around its border).
func averageOpaqueBorderColor(w, h int, at func(x, y int) color.RGBA) (color.RGBA, bool) {
	var rs, gs, bs, n int64
	add := func(x, y int) {
		c := at(x, y)
		if c.A == 0 {
			return
		}
		rs += int64(c.R)
		gs += int64(c.G)
		bs += int64(c.B)
		n++
	}
	for x := 0; x < w; x++ {
		add(x, 0)
		add(x, h-1)
	}
	for y := 0; y < h; y++ {
		add(0, y)
		add(w-1, y)
	}
	if n == 0 {
		return color.RGBA{}, false
	}
	return color.RGBA{R: uint8(rs / n), G: uint8(gs / n), B: uint8(bs / n), A: 255}, true
}

func colorClose(a, b color.RGBA) bool {
	dr := float64(a.R) - float64(b.R)
	dg := float64(a.G) - float64(b.G)
	db := float64(a.B) - float64(b.B)
	dist := math.Sqrt(dr*dr + dg*dg + db*db)
	return dist <= bgColorThreshold
}

// cropRect computes the source rectangle to keep.
func cropRect(src image.Image) image.Rectangle {
	bounds := src.Bounds()
	bbox, hasAlpha := alphaBBox(src)
	if !hasAlpha {
		bbox = bounds
	}

	bw, bh := bbox.Dx(), bbox.Dy()

	// Largest centered 3:4 rectangle that fits inside bbox.
	cw := bw
	ch := int(float64(cw) / targetAspect)
	if ch > bh {
		ch = bh
		cw = int(float64(ch) * targetAspect)
	}

	// For wide/landscape source art, restrict the centroid calculation to a
	// left-anchored square region of the bbox. This set's compositions
	// consistently place the character on/near the left with decorative
	// elements (hair, ribbons, props) flowing out to the right; including
	// those in the centroid pulls the crop off the character's face.
	centroidBBox := bbox
	if bw > int(float64(bh)*1.15) {
		centroidBBox = image.Rect(bbox.Min.X, bbox.Min.Y, min(bbox.Min.X+bh, bbox.Max.X), bbox.Max.Y)
	}
	cx := centroidX(src, centroidBBox)
	x0 := cx - cw/2
	if x0 < bbox.Min.X {
		x0 = bbox.Min.X
	}
	if x0+cw > bbox.Max.X {
		x0 = bbox.Max.X - cw
	}

	// Top-align with a small headroom margin, extending into the source's
	// own transparent padding above the trimmed box when available.
	headroom := ch / 20
	y0 := bbox.Min.Y - headroom
	if y0 < bounds.Min.Y {
		y0 = bounds.Min.Y
	}
	if y0+ch > bounds.Max.Y {
		y0 = bounds.Max.Y - ch
	}

	return image.Rect(x0, y0, x0+cw, y0+ch)
}

// alphaBBox returns the bounding box of pixels with non-zero alpha, and
// whether the image actually has any transparency worth trimming to.
func alphaBBox(img image.Image) (image.Rectangle, bool) {
	b := img.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y
	found := false
	transparentSeen := false

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a == 0 {
				transparentSeen = true
				continue
			}
			found = true
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if !found || !transparentSeen {
		return b, false
	}
	return image.Rect(minX, minY, maxX+1, maxY+1), true
}

// centroidX returns the alpha-weighted horizontal center of mass within
// bbox, used to center the crop on the subject for wide source images.
func centroidX(img image.Image, bbox image.Rectangle) int {
	var sum, weight int64
	for y := bbox.Min.Y; y < bbox.Max.Y; y++ {
		for x := bbox.Min.X; x < bbox.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			w := int64(a >> 8) // 0-255
			sum += int64(x) * w
			weight += w
		}
	}
	if weight == 0 {
		return (bbox.Min.X + bbox.Max.X) / 2
	}
	return int(sum / weight)
}
