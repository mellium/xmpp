// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package attr

import (
	"errors"
	"testing"
)

func TestPublicRandomIDLength(t *testing.T) {
	if s := RandomID(); len(s) != IDLen {
		t.Errorf("Expected length %d got %d", IDLen, len(s))
	}
}

type zeroReader struct{}

func (z zeroReader) Read(b []byte) (n int, err error) {
	for i := range b {
		b[i] = 0
	}

	return len(b), nil
}

func TestRandomIDLength(t *testing.T) {
	for i := 0; i <= 15; i++ {
		if s := randomID(i, zeroReader{}); len(s) != i {
			t.Errorf("Expected length %d got %d", i, len(s))
		}
	}
}

type errorReader struct{}

func (errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("Expected error from error reader")
}

type nopReader struct{}

func (nopReader) Read(p []byte) (int, error) {
	return 0, nil
}

func TestRandomPanicsIfRandReadFails(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected randomID to panic if reading random bytes failed")
		}
	}()
	randomID(16, errorReader{})
}

func TestRandomPanicsIfRandReadWrongLen(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected randomID to panic if no random bytes were read")
		}
	}()
	randomID(16, nopReader{})
}
