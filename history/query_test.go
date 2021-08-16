// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package history_test

import (
	"encoding/xml"
	"testing"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/history"
	"mellium.im/xmpp/internal/xmpptest"
	"mellium.im/xmpp/jid"
)

var (
	_ xml.Unmarshaler     = (*history.Query)(nil)
	_ xml.Marshaler       = (*history.Query)(nil)
	_ xmlstream.Marshaler = (*history.Query)(nil)
	_ xmlstream.WriterTo  = (*history.Query)(nil)
)

var queryEncodingTestCases = []xmpptest.EncodingTestCase{
	0: {
		Value: &history.Query{},
		XML:   `<query xmlns="urn:xmpp:mam:2" queryid=""><x xmlns="jabber:x:data" type="submit"><field type="hidden" var="FORM_TYPE"><value>urn:xmpp:mam:2</value></field></x><set xmlns="http://jabber.org/protocol/rsm"></set></query>`,
	},
	1: {
		Value: &history.Query{
			With: jid.MustParse("example.net"),
		},
		XML: `<query xmlns="urn:xmpp:mam:2" queryid=""><x xmlns="jabber:x:data" type="submit"><field type="hidden" var="FORM_TYPE"><value>urn:xmpp:mam:2</value></field><field type="jid-single" var="with"><value>example.net</value></field></x><set xmlns="http://jabber.org/protocol/rsm"></set></query>`,
	},
	2: {
		Value: &history.Query{
			Start: time.Unix(1, 0).UTC(),
		},
		XML: `<query xmlns="urn:xmpp:mam:2" queryid=""><x xmlns="jabber:x:data" type="submit"><field type="hidden" var="FORM_TYPE"><value>urn:xmpp:mam:2</value></field><field type="text-single" var="start"><value>1970-01-01T00:00:01Z</value></field></x><set xmlns="http://jabber.org/protocol/rsm"></set></query>`,
	},
	3: {
		Value: &history.Query{
			End: time.Unix(1, 0).UTC(),
		},
		XML: `<query xmlns="urn:xmpp:mam:2" queryid=""><x xmlns="jabber:x:data" type="submit"><field type="hidden" var="FORM_TYPE"><value>urn:xmpp:mam:2</value></field><field type="text-single" var="end"><value>1970-01-01T00:00:01Z</value></field></x><set xmlns="http://jabber.org/protocol/rsm"></set></query>`,
	},
	4: {
		Value: &history.Query{
			AfterID: "123",
		},
		XML: `<query xmlns="urn:xmpp:mam:2" queryid=""><x xmlns="jabber:x:data" type="submit"><field type="hidden" var="FORM_TYPE"><value>urn:xmpp:mam:2</value></field><field type="text-single" var="after-id"><value>123</value></field></x><set xmlns="http://jabber.org/protocol/rsm"></set></query>`,
	},
	5: {
		Value: &history.Query{
			BeforeID: "123",
		},
		XML: `<query xmlns="urn:xmpp:mam:2" queryid=""><x xmlns="jabber:x:data" type="submit"><field type="hidden" var="FORM_TYPE"><value>urn:xmpp:mam:2</value></field><field type="text-single" var="before-id"><value>123</value></field></x><set xmlns="http://jabber.org/protocol/rsm"></set></query>`,
	},
	6: {
		Value: &history.Query{
			IDs: []string{"123", "456"},
		},
		XML: `<query xmlns="urn:xmpp:mam:2" queryid=""><x xmlns="jabber:x:data" type="submit"><field type="hidden" var="FORM_TYPE"><value>urn:xmpp:mam:2</value></field><field type="list-multi" var="ids"><value>123</value><value>456</value></field></x><set xmlns="http://jabber.org/protocol/rsm"></set></query>`,
	},
	7: {
		Value: &history.Query{
			Last:    true,
			Reverse: true,
		},
		XML: `<query xmlns="urn:xmpp:mam:2" queryid=""><x xmlns="jabber:x:data" type="submit"><field type="hidden" var="FORM_TYPE"><value>urn:xmpp:mam:2</value></field></x><set xmlns="http://jabber.org/protocol/rsm"><before></before></set><flip-page></flip-page></query>`,
	},
}

func TestEncodeQuery(t *testing.T) {
	xmpptest.RunEncodingTests(t, queryEncodingTestCases)
}
