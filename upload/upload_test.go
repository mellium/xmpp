// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package upload_test

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/upload"
)

var (
	_ xml.Marshaler       = upload.File{}
	_ xml.Unmarshaler     = (*upload.File)(nil)
	_ xmlstream.Marshaler = upload.File{}
	_ xmlstream.WriterTo  = upload.File{}
	_ xml.Marshaler       = upload.Slot{}
	_ xml.Unmarshaler     = (*upload.Slot)(nil)
	_ xmlstream.Marshaler = upload.Slot{}
	_ xmlstream.WriterTo  = upload.Slot{}
)

func mustParseURL(u string) *url.URL {
	uu, err := url.Parse(u)
	if err != nil {
		panic(err)
	}
	return uu
}

var marshalTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: &upload.File{},
		XML:   `<request xmlns="urn:xmpp:http:upload:0" filename="" size="0"></request>`,
	},
	1: {
		Value: &upload.File{Name: "file.opus", Size: 2048, Type: "audio/opus"},
		XML:   `<request xmlns="urn:xmpp:http:upload:0" filename="file.opus" size="2048" content-type="audio/opus"></request>`,
	},
	2: {
		Value: &upload.Slot{},
		XML:   `<slot xmlns="urn:xmpp:http:upload:0"><put url=""></put><get url=""></get></slot>`,
	},
	3: {
		Value: &upload.Slot{
			PutURL: mustParseURL("https://upload.example.net/file.ogg"),
			GetURL: mustParseURL("https://example.net/me/file.ogg"),
		},
		XML: `<slot xmlns="urn:xmpp:http:upload:0"><put url="https://upload.example.net/file.ogg"></put><get url="https://example.net/me/file.ogg"></get></slot>`,
	},
	4: {
		NoUnmarshal: true,
		Value: &upload.Slot{
			Header: http.Header{
				"nonNormalized": []string{"a", "b"},
				"Normalized":    []string{"c"},
				"authorization": []string{"Bearer foo"},
			},
		},
		XML: `<slot xmlns="urn:xmpp:http:upload:0"><put url=""><header name="Authorization">Bearer foo</header></put><get url=""></get></slot>`,
	},
	5: {
		NoUnmarshal: true,
		Value: &upload.Slot{
			Header: http.Header{
				"Cookie": []string{"bar", "baz"},
			},
		},
		XML: `<slot xmlns="urn:xmpp:http:upload:0"><put url=""><header name="Cookie">bar</header><header name="Cookie">baz</header></put><get url=""></get></slot>`,
	},
	6: {
		NoUnmarshal: true,
		Value: &upload.Slot{
			Header: http.Header{
				"Expires": []string{"300"},
			},
		},
		XML: `<slot xmlns="urn:xmpp:http:upload:0"><put url=""><header name="Expires">300</header></put><get url=""></get></slot>`,
	},
	7: {
		NoMarshal: true,
		Value: &upload.Slot{
			Header: http.Header{
				"Cookie": []string{"a", "b"},
			},
		},
		XML: `<slot xmlns="urn:xmpp:http:upload:0"><put url=""><header name="Cookie">a</header><header name="Foo">bar</header><header name="Cookie">b</header></put><get url=""></get></slot>`,
	},
}

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, marshalTestCases)
}
