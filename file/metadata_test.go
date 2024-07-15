package file_test

import (
	"testing"
	"time"

	"mellium.im/xmpp/crypto"
	"mellium.im/xmpp/file"
	"mellium.im/xmpp/internal/xmpptest"
)

var marshalTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: &file.Meta{
			MediaType: "text/plain",
			Date:      time.Date(2024, 01, 01, 01, 01, 01, 00, time.UTC),
			Hash: crypto.HashOutput{
				Hash: crypto.SHA256,
				Out:  []byte{1, 2, 3},
			},
		},
		XML: `<file xmlns="urn:xmpp:file:metadata:0"><media-type>text/plain</media-type><name></name><date>2024-01-01T01:01:01Z</date><size>0</size><hash xmlns="urn:xmpp:hashes:2" algo="sha-256">AQID</hash><width>0</width><height>0</height><length>0</length></file>`,
	},
}

func TestEncode(t *testing.T) {
	xmpptest.RunEncodingTests(t, marshalTestCases)
}
