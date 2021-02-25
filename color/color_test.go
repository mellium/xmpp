// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package color_test

import (
	"image/color"
	"strconv"
	"testing"

	xmppcolor "mellium.im/xmpp/color"
)

func TestSize(t *testing.T) {
	h := xmppcolor.Hash(xmppcolor.None)
	if size := h.Size(); size != 2 {
		t.Errorf("Bad size: want=%d, got=%d", 2, size)
	}
}

var colorTests = [...]struct {
	s     string
	lum   uint8
	cvd   xmppcolor.CVD
	c     color.YCbCr
	panic bool
}{
	0:  {cvd: 4, panic: true},
	1:  {s: "Romeo", lum: 1, c: color.YCbCr{1, 255, 45}},
	2:  {s: "juliet@capulet.lit", lum: 2, c: color.YCbCr{2, 0, 55}},
	3:  {s: "😺", lum: 255, c: color.YCbCr{255, 255, 57}},
	4:  {s: "council", c: color.YCbCr{0, 255, 127}},
	5:  {cvd: xmppcolor.RedGreen, s: "Romeo", c: color.YCbCr{0, 0, 209}},
	6:  {cvd: xmppcolor.RedGreen, s: "juliet@capulet.lit", c: color.YCbCr{0, 255, 199}},
	7:  {cvd: xmppcolor.RedGreen, s: "😺", c: color.YCbCr{0, 0, 197}},
	8:  {cvd: xmppcolor.RedGreen, s: "council", c: color.YCbCr{0, 0, 127}},
	9:  {cvd: xmppcolor.Blue, s: "Romeo", c: color.YCbCr{0, 0, 209}},
	10: {cvd: xmppcolor.Blue, s: "juliet@capulet.lit", c: color.YCbCr{0, 0, 55}},
	11: {cvd: xmppcolor.Blue, s: "😺", c: color.YCbCr{0, 0, 197}},
	12: {cvd: xmppcolor.Blue, s: "council", c: color.YCbCr{0, 0, 127}},
}

func TestHash(t *testing.T) {
	for i, tc := range colorTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			defer func() {
				r := recover()
				switch {
				case r == nil && tc.panic:
					t.Error("No panic detected, but a panic was expected")
				case r != nil && !tc.panic:
					t.Errorf("Unexpected panic detected: %v", r)
				}
			}()

			c := xmppcolor.String(tc.s, tc.lum, tc.cvd)
			if c != tc.c {
				t.Errorf("Invalid color value: want=%v, got=%v", tc.c, c)
			}
		})
	}
}

func BenchmarkHash(b *testing.B) {
	p := []byte("a test string")
	h := xmppcolor.Hash(xmppcolor.None)
	buf := make([]byte, 0, 2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Write(p)
		buf = h.Sum(buf)
		h.Reset()
	}
}

func BenchmarkBytes(b *testing.B) {
	p := []byte("a test string")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xmppcolor.Bytes(p, 128, xmppcolor.None)
	}
}

func BenchmarkString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xmppcolor.String("a test string", 128, xmppcolor.None)
	}
}
