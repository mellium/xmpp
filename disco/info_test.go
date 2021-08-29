// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco_test

import (
	"bytes"
	"encoding/xml"
	"testing"

	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/disco/info"
)

func TestMarshalQuery(t *testing.T) {
	const expected = `<query xmlns="http://jabber.org/protocol/disco#info" node="test"></query>`
	query := disco.InfoQuery{Node: "test"}
	t.Run("Marshal", func(t *testing.T) {
		b, err := xml.Marshal(query)
		if err != nil {
			t.Fatalf("unexpected error marshaling query: %v", err)
		}
		if !bytes.Equal(b, []byte(expected)) {
			t.Fatalf("wrong output:\nwant=%s,\n got=%s", expected, b)
		}
	})
	t.Run("Encode", func(t *testing.T) {
		var buf bytes.Buffer
		e := xml.NewEncoder(&buf)
		_, err := query.WriteXML(e)
		if err != nil {
			t.Fatalf("unexpected error marshaling query: %v", err)
		}
		err = e.Flush()
		if err != nil {
			t.Fatalf("unexpected error flushing: %v", err)
		}
		if out := buf.String(); out != expected {
			t.Fatalf("wrong output:\nwant=%s,\n got=%s", expected, out)
		}
	})
}

func TestUnmarshal(t *testing.T) {
	const infoXML = `<query node="test" xmlns='http://jabber.org/protocol/disco#info'>
  <identity
      category='conference'
      type='text'
      name='Play-Specific Chatrooms'
			xml:lang='en'/>
  <identity
      category='directory'
      type='chatroom'
      name='Play-Specific Chatrooms'/>
  <feature var='http://jabber.org/protocol/disco#info'/>
  <feature var='http://jabber.org/protocol/disco#items'/>
  <feature var='http://jabber.org/protocol/muc'/>
  <x xmlns='jabber:x:data' type='result'>
    <field var='FORM_TYPE' type='hidden'>
      <value>http://jabber.org/network/serverinfo</value>
    </field>
    <field var='c2s_port'>
      <value>5222</value>
    </field>
  </x>
</query>`
	var infoResp disco.Info
	err := xml.Unmarshal([]byte(infoXML), &infoResp)
	if err != nil {
		t.Fatalf("unexpected error unmarshaling: %v", err)
	}
	if infoResp.Node != "test" {
		t.Errorf("node did not unmarshal correctly: want=test, got=%s", infoResp.Node)
	}
	if l := len(infoResp.Identity); l != 2 {
		t.Errorf("wrong number of identities: want=2, got=%d", l)
	}
	ident := info.Identity{
		XMLName:  xml.Name{Space: disco.NSInfo, Local: "identity"},
		Category: "conference",
		Type:     "text",
		Name:     "Play-Specific Chatrooms",
		Lang:     "en",
	}
	if ident != infoResp.Identity[0] {
		t.Errorf("wrong identity: want=%v, got=%v", ident, infoResp.Identity[0])
	}
	if l := len(infoResp.Features); l != 3 {
		t.Errorf("wrong number of features: want=3, got=%d", l)
	}
	if v := infoResp.Features[0].Var; v != disco.NSInfo {
		t.Errorf("wrong first feature: want=%s, got=%s", disco.NSInfo, v)
	}
	if infoResp.Form == nil {
		t.Errorf("form was not unmarshaled")
	}
	const serverInfo = "http://jabber.org/network/serverinfo"
	if s, ok := infoResp.Form[0].GetString("FORM_TYPE"); !ok || s != serverInfo {
		t.Errorf("wrong value for FORM_TYPE: want=%s, got=%s", serverInfo, s)
	}
	if s, ok := infoResp.Form[0].GetString("c2s_port"); !ok || s != "5222" {
		t.Errorf("wrong value for FORM_TYPE: want=5222, got=%s", s)
	}
}
