// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"encoding/xml"
	"testing"

	"bitbucket.org/mellium/xmpp/jid"
)

var (
	validAttrs = []xml.Attr{
		{xml.Name{"", "id"}, "1234"},
		{xml.Name{"", "version"}, "1.0"},
		{xml.Name{"", "to"}, "shakespeare.lit"},
		{xml.Name{"", "from"}, "prospero@shakespeare.lit"},
		{xml.Name{"xmlns", "stream"}, "http://etherx.jabber.org/streams"},
		{xml.Name{"xml", "lang"}, "en"},
		{xml.Name{"", "xmlns"}, "jabber:client"},
	}
	validName = xml.Name{"stream", "stream"}
)

// FromStartElement should validate attributes.
func TestStreamFromStartElement(t *testing.T) {
	var data = []struct {
		start       xml.StartElement
		shouldError bool
	}{
		{xml.StartElement{validName, validAttrs}, false},
		{xml.StartElement{xml.Name{"stream", "wrong"}, validAttrs}, true},
		{xml.StartElement{xml.Name{"wrong", "stream"}, validAttrs}, true},
		{xml.StartElement{validName, []xml.Attr{
			{xml.Name{"", "id"}, "1234"},
			{xml.Name{"", "version"}, "1.0"},
			{xml.Name{"", "to"}, "shakespeare.lit"},
			{xml.Name{"", "from"}, "prospero@shakespeare.lit"},
			{xml.Name{"xmlns", "stream"}, "http://etherx.jabber.org/streams"},
			{xml.Name{"xml", "lang"}, "en"},
			{xml.Name{"", "xmlns"}, "jabber:wrong"},
		}}, true},
		{xml.StartElement{validName, []xml.Attr{
			{xml.Name{"", "id"}, "1234"},
			{xml.Name{"", "version"}, "1.0"},
			{xml.Name{"", "to"}, "shakespeare.lit"},
			{xml.Name{"", "from"}, "prospero@shakespeare.lit"},
			{xml.Name{"xmlns", "stream"}, "urn:jabber:wrong"},
			{xml.Name{"xml", "lang"}, "en"},
			{xml.Name{"", "xmlns"}, "jabber:client"},
		}}, true},
	}

	for _, d := range data {
		if _, err := FromStartElement(d.start); (err != nil) != d.shouldError {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestStreamStartElement(t *testing.T) {
	to, _ := jid.ParseString("shakespeare.lit")
	from, _ := jid.ParseString("example.net")

	// The default stream namespace should be jabber:client.
	s := New(to, from)
	se := s.StartElement()
	xmlnsname := xml.Name{"", "xmlns"}
	for _, attr := range se.Attr {
		if attr.Name == xmlnsname && attr.Value != "jabber:client" {
			t.Log("Default stream xmlns should be jabber:client.")
			t.Fail()
		}
	}

	// Server to Server streams should have jabber:server namespace.
	se = New(to, from, ServerToServer).StartElement()
	for _, attr := range se.Attr {
		if attr.Name == xmlnsname && attr.Value != "jabber:server" {
			t.Log("ServerToServer option should set namespace to jabber:server.")
			t.Fail()
		}
	}

	// The NoVersion option should result in no stream version attribute.
	se = New(to, from, NoVersion).StartElement()
	for _, attr := range se.Attr {
		if attr.Name.Local == "version" {
			t.Log("NoVersion option should result in no version attribute.")
			t.Fail()
		}
	}
}
