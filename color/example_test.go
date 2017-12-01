// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package color_test

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"golang.org/x/image/font"
	_ "golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"

	colorgen "mellium.im/xmpp/color"
)

// mix a foreground color with a background color ignoring the alpha channel.
func mix(fg color.Color, bg color.Color) color.Color {
	const (
		factor = 0.4
		inv    = 1 - factor
		maxu16 = 1<<16 - 1
	)
	rb, gb, bb, _ := bg.RGBA()
	rf, gf, bf, _ := fg.RGBA()

	return color.RGBA{
		R: uint8(factor*float32(maxu16-rb) + inv*float32(rf)),
		G: uint8(factor*float32(maxu16-gb) + inv*float32(gf)),
		B: uint8(factor*float32(maxu16-bb) + inv*float32(bf)),
		A: 255,
	}
}

func Example() {
	strs := []string{
		"Beautiful",
		"Catchup",
		"Dandelion",
		"Fuego Borrego",
		"Green Giant",
		"Mailman",
		"Papa Shrimp",
		"Pockets",
		"Spoon Foot",
		"Sunshine",
		"Thespian",
		"Twinkle Toes",
		"Zodiac",
	}

	img := image.NewRGBA(image.Rect(0, 0, 300, 216))
	parts := []color.Color{color.Black, color.White}

	for x, bg := range parts {
		bounds := img.Bounds()
		w := bounds.Max.X / len(parts)
		bounds.Min.X = w * x
		bounds.Max.X = w * (x + 1)
		draw.Draw(img, bounds, &image.Uniform{bg}, image.ZP, draw.Src)

		for y, s := range strs {
			d := &font.Drawer{
				Dst: img,
				Src: image.NewUniform(mix(
					colorgen.String(s, 128, colorgen.None),
					bg,
				)),
				Face: inconsolata.Regular8x16,
				Dot: fixed.Point26_6{
					fixed.Int26_6((12 + bounds.Min.X) * 64),
					fixed.Int26_6(16 * (y + 1) * 64),
				},
			}

			d.DrawString(s)
		}
	}

	f, err := os.Create("gonicks.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
