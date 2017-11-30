// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package color implements XEP-0392: Consistent Color Generation.
package color

import (
	"crypto/sha1"
	"encoding/binary"
	"hash"
	"image/color"
	"math"
)

// The size of the hash output.
const Size = 2

// A list of color vision deficiencies.
const (
	None uint8 = iota
	RedGreen
	Blue
)

// Hash returns a new hash.Hash computing the Y'CbCr color.
// For more information see Sum.
func Hash(cvd uint8) hash.Hash {
	return digest{
		Hash: sha1.New(),
		cvd:  cvd,
	}
}

type digest struct {
	hash.Hash
	cvd uint8
}

func (d digest) Size() int { return Size }
func (d digest) Sum(b []byte) []byte {
	b = d.Hash.Sum(b)
	i := binary.LittleEndian.Uint16(b[:2])
	switch d.cvd {
	case None:
	case RedGreen:
		i &= 0x7fff
	case Blue:
		i = (i & 0x7fff) | (((i & 0x4000) << 1) ^ 0x8000)
	default:
		panic("color: invalid color vision deficiency")
	}
	angle := float64(i) / 65536 * 2 * math.Pi
	cr, cb := math.Sincos(angle)
	factor := 0.5 / math.Max(math.Abs(cr), math.Abs(cb))
	cb, cr = cb*factor, cr*factor

	b[0] = uint8(math.Min(math.Max(cb+0.5, 0)*255, 255))
	b[1] = uint8(math.Min(math.Max(cr+0.5, 0)*255, 255))
	return b[:Size]
}

// Sum returns a color in the Y'CbCr colorspace in the form [Cb, Cr] that is
// consistent for the same inputs.
//
// If a color vision deficiency constant is provided (other than None), the
// algorithm attempts to avoid confusable colors.
func Sum(data []byte, cvd uint8) [Size]byte {
	b := make([]byte, 0, Size)
	h := Hash(cvd)
	h.Write(data)
	b = h.Sum(b)
	return [Size]byte{b[0], b[1]}
}

// Bytes converts a byte slice to a color.YCbCr.
// The recommended luma value is 255 (Y'=0.5, Y=0.732).
//
// For more information see Sum.
func Bytes(b []byte, luma uint8, cvd uint8) color.YCbCr {
	ba := Sum(b, cvd)
	return color.YCbCr{
		Y:  luma,
		Cb: ba[0],
		Cr: ba[1],
	}
}

// String converts a string to a color.YCbCr.
// The recommended luma value is 255 (Y'=0.5, Y=0.732).
//
// For more information see Sum.
func String(s string, luma uint8, cvd uint8) color.YCbCr {
	return Bytes([]byte(s), luma, cvd)
}
