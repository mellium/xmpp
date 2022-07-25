// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package cid implements content-ID URLs.
package cid // import "mellium.im/xmpp/internal/cid"

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
)

// Parse parses a raw url into a URL structure.
//
// The URL may or may not include the cid: scheme, but will fail if the exact
// structure for CID URLs is not met (or if the scheme is present and is
// anything other than "cid").
func Parse(rawURL string) (*URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	var opaque string
	switch u.Scheme {
	case "":
		opaque = u.Path
	case "cid":
		opaque = u.Opaque
	default:
		return nil, fmt.Errorf("cid: failed to parse URL with scheme %q", u.Scheme)
	}
	if opaque == "" {
		return nil, errors.New("cid: URL is invalid and resulted in empty CID")
	}
	var plusIdx, atIdx int
opaque:
	for i, b := range opaque {
		switch b {
		case '+':
			plusIdx = i
		case '@':
			if plusIdx <= 0 {
				return nil, errors.New("cid: missing hash name")
			}
			atIdx = i
			break opaque
		}
	}
	if atIdx <= 0 || atIdx == len(opaque)-1 {
		return nil, errors.New("cid: missing domain part")
	}

	out := &URL{
		HashName: opaque[:plusIdx],
		Domain:   opaque[atIdx+1:],
	}
	hash, err := hex.DecodeString(opaque[plusIdx+1 : atIdx])
	if err != nil {
		return nil, fmt.Errorf("cid: error decoding hash: %w", err)
	}
	out.Hash = hash
	if len(hash) == 0 {
		return nil, errors.New("cid: no hash found")
	}

	return out, nil
}

// A URL represents a parsed CID URL.
type URL struct {
	HashName string
	Domain   string
	Hash     []byte
}

// String reassembles the URL into a valid URL string.
func (u *URL) String() string {
	return fmt.Sprintf("cid:%s+%x@%s", u.HashName, u.Hash, u.Domain)
}
