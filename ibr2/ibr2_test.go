// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibr2

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"reflect"
	"testing"
)

// TestList checks that the server listing is generated properly and does not
// repeat challenge types.
func TestList(t *testing.T) {
	b := new(bytes.Buffer)
	d := xml.NewDecoder(b)
	e := xml.NewEncoder(b)
	f := Recovery(
		Challenge{Type: "jabber:x:data"},
		Challenge{Type: "pow"},
		Challenge{Type: "jabber:x:data"})
	_, err := f.List(context.Background(), e, xml.StartElement{Name: xml.Name{Local: "recover"}})
	if err != nil {
		t.Fatalf("List returned error: %v\n", err)
	}
	o := struct {
		XMLName   xml.Name `xml:"recover"`
		Challenge []string `xml:"challenge"`
	}{}
	err = d.Decode(&o)
	if err != nil {
		t.Fatalf("Decoding error: %v\n", err)
	}
	if len(o.Challenge) != 2 {
		t.Fatalf("Expected 2 challenges, got %d", len(o.Challenge))
	}
	if o.Challenge[0] != "jabber:x:data" {
		t.Errorf("Expected first challenge to be jabber:x:data but got %s", o.Challenge[0])
	}
	if o.Challenge[1] != "pow" {
		t.Errorf("Expected second challenge to be pow but got %s", o.Challenge[1])
	}
}

var parseTests = [...]struct {
	Listing    []string
	Challenges []string
	Supported  bool
}{
	0: {
		[]string{"test", "test", "test", "test", "test", "test"},
		[]string{"type", "more", "test"},
		true,
	},
	1: {
		[]string{"test", "test", "test", "test", "test", "test"},
		[]string{"type", "more"},
		false,
	},
	2: {
		[]string{"test", "test"},
		[]string{"type", "more", "test"},
		true,
	},
	3: {
		[]string{"test", "test"},
		[]string{"type", "more", "new", "castle"},
		false,
	},
	4: {
		[]string{"a", "new", "test"},
		[]string{"new", "test", "a"},
		true,
	},
	5: {
		[]string{},
		[]string{"new", "test", "a"},
		true,
	},
	6: {
		[]string{"nope", "never"},
		[]string{},
		false,
	},
	7: {
		[]string{},
		[]string{},
		true,
	},
}

// TestParse checks that clients parse challenge feature listings correctly and
// that they correctly determine if they support all the listed challenge types.
func TestParse(t *testing.T) {
	for i, tc := range parseTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			// Create the feature with the named challenges.
			challenges := make([]Challenge, len(tc.Challenges))
			for i, c := range tc.Challenges {
				challenges[i] = Challenge{Type: c}
			}
			r := Register(challenges...)

			// Marshal an XML listing for us to decode.
			b, err := xml.Marshal(struct {
				XMLName    xml.Name `xml:"urn:xmpp:register:0 recovery"`
				Challenges []string `xml:"challenge"`
			}{
				Challenges: tc.Listing,
			})
			if err != nil {
				t.Fatal(err)
			}

			d := xml.NewDecoder(bytes.NewReader(b))
			tok, err := d.Token()
			if err != nil {
				t.Fatal(err)
			}
			start, ok := tok.(xml.StartElement)
			if !ok {
				t.Fatalf("Marshaled bad XML; didn't get start element, got %#v", tok)
			}
			req, data, err := r.Parse(context.Background(), d, &start)

			supported, ok := data.(bool)
			switch {
			case req:
				t.Error("Feature parsed as required")
			case err != nil:
				t.Errorf("Unexpected error while parsing feature: %v", err)
			case !ok:
				t.Errorf("Parse returned wrong type data; want=bool, got=%v", reflect.TypeOf(supported))
			case supported != tc.Supported:
				t.Errorf("Parse got mismatched feature support: want=%v, got=%v", tc.Supported, supported)
			}

		})
	}
}
