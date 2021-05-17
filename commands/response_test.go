// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package commands_test

import (
	"encoding/xml"
	"testing"

	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/internal/xmpptest"
)

func TestNoteTypes(t *testing.T) {
	xmpptest.RunEncodingTests(t, []xmpptest.EncodingTestCase{
		{
			Value: &struct {
				XMLName xml.Name          `xml:"foo"`
				Type    commands.NoteType `xml:"notetype,attr"`
			}{
				XMLName: xml.Name{Local: "foo"},
			},
			XML: `<foo notetype="info"></foo>`,
		},
		{
			Value: &struct {
				XMLName xml.Name          `xml:"foo"`
				Type    commands.NoteType `xml:"notetype,attr"`
			}{
				XMLName: xml.Name{Local: "foo"},
				Type:    commands.NoteWarn,
			},
			XML: `<foo notetype="warn"></foo>`,
		},
		{
			Value: &struct {
				XMLName xml.Name          `xml:"foo"`
				Type    commands.NoteType `xml:"notetype,attr"`
			}{
				XMLName: xml.Name{Local: "foo"},
				Type:    commands.NoteError,
			},
			XML: `<foo notetype="error"></foo>`,
		},
	})
}
