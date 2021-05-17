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

func TestNotes(t *testing.T) {
	xmpptest.RunEncodingTests(t, []xmpptest.EncodingTestCase{
		{
			Value: &commands.Note{XMLName: xml.Name{Local: "note"}},
			XML:   `<note type="info"></note>`,
		},
		{
			Value: &commands.Note{XMLName: xml.Name{Local: "note"}, Type: commands.NoteError, Value: "foo"},
			XML:   `<note type="error">foo</note>`,
		},
		{
			Value:       &commands.Note{XMLName: xml.Name{Local: "note"}, Type: commands.NoteType(5), Value: "foo"},
			XML:         `<note type="NoteType(5)">foo</note>`,
			NoUnmarshal: true,
		},
	})
}
