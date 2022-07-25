// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package bin_test

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/bin"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

var (
	_ xml.Marshaler       = (*bin.Data)(nil)
	_ xml.Unmarshaler     = (*bin.Data)(nil)
	_ xmlstream.Marshaler = (*bin.Data)(nil)
	_ xmlstream.WriterTo  = (*bin.Data)(nil)
)

var dataEncodingTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: &bin.Data{},
		XML:   `<data xmlns="urn:xmpp:bob"></data>`,
	},
	1: {
		Value: &bin.Data{
			CID:    "test",
			Type:   "application/octet-stream",
			MaxAge: 3 * time.Second,
			Data:   []byte{1, 2, 3},
		},
		XML: `<data xmlns="urn:xmpp:bob" type="application/octet-stream" max-age="3" cid="test">AQID</data>`,
	},
	// TODO: the next two tests marshal to the same output. I don't like that
	// there is not a one-to-one mapping between the data structure and XML
	// output, but the solution of including a NoCache field was the best thing
	// that I could think of with Go's limited type system.
	// Making the MaxAge field nilable would work, but makes it much harder to
	// work with too. Other suggestions welcome before 1.0.
	2: {
		Value: &bin.Data{
			MaxAge:  3 * time.Second,
			NoCache: true,
		},
		XML:         `<data xmlns="urn:xmpp:bob" max-age="0"></data>`,
		NoUnmarshal: true,
	},
	3: {
		Value: &bin.Data{
			NoCache: true,
		},
		XML: `<data xmlns="urn:xmpp:bob" max-age="0"></data>`,
	},
	4: {
		// Check that any base64 decoding errors are handled.
		Value:     &bin.Data{},
		Err:       base64.CorruptInputError(1),
		XML:       `<data xmlns="urn:xmpp:bob">a=</data>`,
		NoMarshal: true,
	},
}

func TestDataEncoding(t *testing.T) {
	xmpptest.RunEncodingTests(t, dataEncodingTestCases)
}

var handlerTestCases = []struct {
	H    bin.Handler
	Err  error
	Data *bin.Data
}{
	{Data: &bin.Data{}, Err: stanza.Error{Condition: stanza.ItemNotFound}},
	{H: bin.Handler{
		Get: func(string) (*bin.Data, error) {
			return &bin.Data{CID: "test", Data: []byte{1, 2, 3}}, nil
		},
	}, Data: &bin.Data{CID: "test", Data: []byte{1, 2, 3}}},
}

func TestHandler(t *testing.T) {
	for i, tc := range handlerTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			m := mux.New(
				stanza.NSServer,
				bin.Handle(tc.H),
			)
			cs := xmpptest.NewClientServer(xmpptest.ClientHandler(m))
			data, err := bin.Get(context.Background(), cs.Server, cs.Client.LocalAddr(), "test")
			if !errors.Is(err, tc.Err) {
				t.Fatalf("wrong error: want=%v, got=%v", tc.Err, err)
			}
			if !reflect.DeepEqual(data, tc.Data) {
				t.Errorf("wrong data: want=%+v, got=%+v", tc.Data, data)
			}
		})
	}
}
