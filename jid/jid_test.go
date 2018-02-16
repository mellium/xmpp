// Copyright 2015 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package jid_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net"
	"strings"
	"testing"

	"mellium.im/xmpp/jid"
)

// Compile time checks to make sure that JID and *jid.JID match several interfaces.
var (
	_ fmt.Stringer        = (*jid.JID)(nil)
	_ xml.MarshalerAttr   = (*jid.JID)(nil)
	_ xml.UnmarshalerAttr = (*jid.JID)(nil)
	_ xml.Marshaler       = (*jid.JID)(nil)
	_ xml.Unmarshaler     = (*jid.JID)(nil)
	_ net.Addr            = (*jid.JID)(nil)
)

func TestValidJIDs(t *testing.T) {
	for i, tc := range [...]struct {
		jid, lp, dp, rp string
	}{
		0: {"example.net", "", "example.net", ""},
		1: {"example.net/rp", "", "example.net", "rp"},
		2: {"mercutio@example.net", "mercutio", "example.net", ""},
		3: {"mercutio@example.net/rp", "mercutio", "example.net", "rp"},
		4: {"mercutio@example.net/rp@rp", "mercutio", "example.net", "rp@rp"},
		5: {"mercutio@example.net/rp@rp/rp", "mercutio", "example.net", "rp@rp/rp"},
		6: {"mercutio@example.net/@", "mercutio", "example.net", "@"},
		7: {"mercutio@example.net//@", "mercutio", "example.net", "/@"},
		8: {"mercutio@example.net//@//", "mercutio", "example.net", "/@//"},
		9: {"[::1]", "", "[::1]", ""},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			j, err := jid.Parse(tc.jid)
			if err != nil {
				t.Fatal(err)
			}
			if j.Domainpart() != tc.dp {
				t.Errorf("Got domainpart %s but expected %s", j.Domainpart(), tc.dp)
			}
			if j.Localpart() != tc.lp {
				t.Errorf("Got localpart %s but expected %s", j.Localpart(), tc.lp)
			}
			if j.Resourcepart() != tc.rp {
				t.Errorf("Got resourcepart %s but expected %s", j.Resourcepart(), tc.rp)
			}
		})
	}
}

var invalidutf8 = string([]byte{0xff, 0xfe, 0xfd})

func TestInvalidParseJIDs(t *testing.T) {
	for i, tc := range [...]string{
		0:  "test@/test",
		1:  invalidutf8 + "@example.com/rp",
		2:  invalidutf8 + "/rp",
		3:  invalidutf8,
		4:  "example.com/" + invalidutf8,
		5:  "lp@/rp",
		6:  `b"d@example.net`,
		7:  `b&d@example.net`,
		8:  `b'd@example.net`,
		9:  `b:d@example.net`,
		10: `b<d@example.net`,
		11: `b>d@example.net`,
		12: `e@example.net/`,
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := jid.Parse(tc)
			if err == nil {
				t.Errorf("Expected JID %s to fail", tc)
			}
		})
	}
}

func TestInvalidNewJIDs(t *testing.T) {
	for i, tc := range [...]struct {
		lp, dp, rp string
	}{
		0: {strings.Repeat("a", 1024), "example.net", ""},
		1: {"e", "example.net", strings.Repeat("a", 1024)},
		2: {"b/d", "example.net", ""},
		3: {"b@d", "example.net", ""},
		4: {"e", "[example.net]", ""},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := jid.New(tc.lp, tc.dp, tc.rp)
			if err == nil {
				t.Errorf("Expected composition of JID parts %s to fail", tc)
			}
		})
	}
}

func TestMarshalAttrEmpty(t *testing.T) {
	attr, err := ((*jid.JID)(nil)).MarshalXMLAttr(xml.Name{})
	switch {
	case err != nil:
		t.Logf("Marshaling an empty JID to an attr should not error but got %v\n", err)
		t.Fail()
	case attr != xml.Attr{}:
		t.Logf("Error marshaling empty JID expected Attr{} but got: %+v\n", err)
		t.Fail()
	}
}

func TestMustParsePanics(t *testing.T) {
	for i, tc := range [...]struct {
		jid         string
		shouldPanic bool
	}{
		0: {"@me", true},
		1: {"@`me", true},
		2: {"e@example.net", false},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			defer func() {
				r := recover()
				switch {
				case tc.shouldPanic && r == nil:
					t.Error("Must parse should panic on invalid JID")
				case !tc.shouldPanic && r != nil:
					t.Error("Must parse should not panic on valid JID")
				}
			}()
			jid.MustParse(tc.jid)
		})
	}
}

func TestEqual(t *testing.T) {
	m := jid.MustParse("mercutio@example.net/test")
	for i, tc := range [...]struct {
		j1, j2 *jid.JID
		eq     bool
	}{
		0: {m, jid.MustParse("mercutio@example.net/test"), true},
		1: {m.Bare(), jid.MustParse("mercutio@example.net"), true},
		2: {m.Domain(), jid.MustParse("example.net"), true},
		3: {m, jid.MustParse("mercutio@example.net/nope"), false},
		4: {m, jid.MustParse("mercutio@e.com/test"), false},
		5: {m, jid.MustParse("m@example.net/test"), false},
		6: {(*jid.JID)(nil), (*jid.JID)(nil), true},
		7: {m, (*jid.JID)(nil), false},
		8: {(*jid.JID)(nil), m, false},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			switch {
			case tc.eq && !tc.j1.Equal(tc.j2):
				t.Errorf("JIDs %s and %s should be equal", tc.j1, tc.j2)
			case !tc.eq && tc.j1.Equal(tc.j2):
				t.Errorf("JIDs %s and %s should not be equal", tc.j1, tc.j2)
			}
		})
	}
}

func TestNetwork(t *testing.T) {
	if jid.MustParse("test").Network() != "xmpp" {
		t.Error("Network should be `xmpp`")
	}
}

func TestCopy(t *testing.T) {
	m := jid.MustParse("mercutio@example.net/test")
	m2 := m.Copy()
	switch {
	case !m.Equal(m2):
		t.Error("Copying a JID should still result in equal JIDs")
	case m == m2:
		t.Error("Copying a JID should result in a different JID pointer")
	}
}

func TestWithResource(t *testing.T) {
	for i, tc := range [...]struct {
		jid string
		res string
		err bool
	}{
		0: {"mercutio@example.net/test", "new", false},
		1: {"mercutio@example.net/test", invalidutf8, true},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			old := jid.MustParse(tc.jid)
			new, err := old.WithResource(tc.res)
			switch {
			case (err != nil) && !tc.err:
				t.Fatal("Unexpected error", err)
			case tc.err:
				return
			case old == new:
				t.Fatal("Expected different pointers for JID copy")
			}
			if old.String() != tc.jid {
				t.Fatalf("WithResource should clone data")
			}
			if r := new.Resourcepart(); r != tc.res {
				t.Errorf("Unexpected resourcepart: want=`%s', got=`%s'", tc.res, r)
			}
			if new.Domainpart() != old.Domainpart() {
				t.Errorf("Unexpected domainpart mutation: want=`%s', got=`%s'", old.Domainpart(), new.Domainpart())
			}
			if new.Localpart() != old.Localpart() {
				t.Errorf("Unexpected localpart mutation: want=`%s', got=`%s'", old.Localpart(), new.Localpart())
			}
		})
	}
}

func TestMarshalXML(t *testing.T) {
	// Test default marshaling
	j := jid.MustParse("feste@shakespeare.lit")
	b, err := xml.Marshal(j)
	switch expected := `<JID>feste@shakespeare.lit</JID>`; {
	case err != nil:
		t.Error(err)
	case string(b) != expected:
		t.Errorf("Error marshaling JID, expected `%s` but got `%s`", expected, string(b))
	}

	// Test encoding with a custom element
	j = jid.MustParse("feste@shakespeare.lit/ilyria")
	var buf bytes.Buffer
	start := xml.StartElement{Name: xml.Name{Space: "", Local: "item"}, Attr: []xml.Attr{}}
	e := xml.NewEncoder(&buf)
	err = e.EncodeElement(j, start)
	switch expected := `<item>feste@shakespeare.lit/ilyria</item>`; {
	case err != nil:
		t.Error(err)
	case buf.String() != expected:
		t.Errorf("Error encoding JID, expected `%s` but got `%s`", expected, buf.String())
	}

	// Test encoding a nil JID
	j = (*jid.JID)(nil)
	b, err = xml.Marshal(j)
	switch expected := ``; {
	case err != nil:
		t.Error(err)
	case string(b) != expected:
		t.Errorf("Error marshaling JID, expected `%s` but got `%s`", expected, string(b))
	}
}

func TestUnmarshal(t *testing.T) {
	for i, test := range [...]struct {
		xml string
		jid *jid.JID
		err bool
	}{
		0: {`<item>feste@shakespeare.lit/ilyria</item>`, jid.MustParse("feste@shakespeare.lit/ilyria"), false},
		1: {`<jid>feste@shakespeare.lit</jid>`, jid.MustParse("feste@shakespeare.lit"), false},
		2: {`<oops>feste@shakespeare.lit</bad>`, nil, true},
		3: {`<item></item>`, nil, true},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			j := &jid.JID{}
			err := xml.Unmarshal([]byte(test.xml), j)
			switch {
			case test.err && err == nil:
				t.Errorf("Expected unmarshaling `%s` as a JID to return an error", test.xml)
			case !test.err && err != nil:
				t.Error("Unexpected error:", err)
			case err != nil:
				return
			case !test.jid.Equal(j):
				t.Errorf("Expected JID to unmarshal to `%s` but got `%s`", test.jid, j)
			}
		})
	}
}

func TestString(t *testing.T) {
	for i, tc := range [...]string{
		0: "example.com",
		1: "feste@example.com",
		2: "feste@example.com/testabc",
		3: "example.com/test",
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			j := jid.MustParse(tc)

			// Check that String() and jid.Parse() are inverse operations
			if js := j.String(); js != tc {
				t.Errorf("want=%s, got=%s", tc, js)
			}

			// Check that String() does not allocate

			// If the code is instrumented for coverage, allocations that happen there
			// break this test. This is annoying, but I'm not sure of a better way to
			// fix it.
			var okallocs float64
			if testing.CoverMode() != "" {
				okallocs = 3.0
			}

			if n := testing.AllocsPerRun(1000, func() { _ = j.String() }); n > okallocs {
				t.Errorf("got %f allocs, want %f", n, okallocs)
			}
		})
	}
}

// Malloc tests may be flakey under GCC until it improves its escape analysis.

func TestSplitMallocs(t *testing.T) {
	n := testing.AllocsPerRun(1000, func() {
		jid.SplitString("olivia@example.net/ilyria")
	})
	if n > 0 {
		t.Errorf("got %f allocs, want 0", n)
	}
}
