// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"encoding/xml"
	"fmt"
	"testing"

	"mellium.im/xmpp"
)

type muxTestCase struct {
	panic   bool
	m       []pair
	name    xml.Name
	pattern string
}

type pair struct {
	pattern string
	handler func(*testing.T) xmpp.Handler
}

func failHandler(t *testing.T) xmpp.Handler {
	return xmpp.HandlerFunc(func(_ xml.TokenReader, _ *xml.StartElement) error {
		t.Errorf("Got the wrong handler")
		return nil
	})
}

func passHandler(_ *testing.T) xmpp.Handler {
	return xmpp.HandlerFunc(func(_ xml.TokenReader, _ *xml.StartElement) error {
		return nil
	})
}

func nilHandler(_ *testing.T) xmpp.Handler {
	return nil
}

var muxTestCases = [...]muxTestCase{
	0: {panic: true, m: []pair{{"local", nilHandler}}},
	1: {panic: true, m: []pair{{"", failHandler}}},
	2: {panic: true, m: []pair{
		{"local", failHandler},
		{"local", failHandler},
	}},
	3: {
		m: []pair{
			{"space local", passHandler},
			{"local", failHandler},
			{"space", failHandler},
		},
		name:    xml.Name{Space: "space", Local: "local"},
		pattern: "space local",
	},
	4: {
		m: []pair{
			{"space ", passHandler},
			{"local", failHandler},
			{"space", failHandler},
		},
		name:    xml.Name{Space: "space", Local: ""},
		pattern: "space ",
	},
	5: {
		m: []pair{
			{"local", failHandler},
			{" local", passHandler},
		},
		name:    xml.Name{Space: "", Local: "local"},
		pattern: " local",
	},
	// TODO: Unmatched pattern should return default handler.
	// (also find a way to register a replacement default handler… empty name?)
	//6: {
	//	name:    xml.Name{Space: "", Local: "local"},
	//	pattern: "",
	//},
}

func TestMux(t *testing.T) {
	for i, tc := range muxTestCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			defer func() {
				r := recover()
				switch {
				case r != nil && !tc.panic:
					t.Errorf("Unexpected panic: %v", r)
				case r == nil && tc.panic:
					t.Errorf("Expected test to panic")
				}
			}()

			mux := &xmpp.ServeMux{}
			for _, p := range tc.m {
				h := p.handler(t)
				if f, ok := h.(xmpp.HandlerFunc); ok {
					mux.HandleFunc(p.pattern, f)
				} else {
					mux.Handle(p.pattern, h)
				}
			}

			h, pattern := mux.Handler(tc.name)
			if pattern != tc.pattern {
				t.Errorf("Got wrong pattern: want=`%v', got=`%v'", tc.pattern, pattern)
			}
			if h == nil {
				t.Fatal("Got nil handler")
			}
			h.HandleXMPP(nil, nil)
		})
	}
}
