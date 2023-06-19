// Copyright 2023 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package crypto

import (
	"encoding/base64"
	"encoding/xml"
	"errors"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
)

// A Key is an opaque collection of bytes (the actual key format will depend on
// the encryption type being used).
// The key may be trusted or distrusted (the default).
type Key struct {
	Trusted bool
	KeyID   []byte
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (k Key) TokenReader() xml.TokenReader {
	var local string
	if k.Trusted {
		local = "trust"
	} else {
		local = "distrust"
	}
	data := make([]byte, base64.StdEncoding.EncodedLen(len(k.KeyID)))
	base64.StdEncoding.Encode(data, k.KeyID)
	return xmlstream.Wrap(
		xmlstream.Token(xml.CharData(data)),
		xml.StartElement{
			Name: xml.Name{Local: local},
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (k Key) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, k.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (k Key) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := k.WriteXML(e)
	return err
}

var errTrustElement = errors.New("expected trust or distrust element only")

// UnmarshalXML implements xml.Unmarshaler.
func (k *Key) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	k.Trusted = start.Name.Local == "trust"
	if !k.Trusted && start.Name.Local != "distrust" {
		return errTrustElement
	}

	trust := struct {
		// Use innerxml instead of chardata to make sure we consume the entire
		// element and if anything that's not base64 encoded has been smuggled into
		// it somehow we have an error on decoding.
		Inner []byte `xml:",innerxml"`
	}{}
	err := d.DecodeElement(&trust, &start)
	if err != nil {
		return err
	}
	expectedLen := base64.StdEncoding.DecodedLen(len(trust.Inner))
	if len(k.KeyID) < expectedLen {
		k.KeyID = make([]byte, expectedLen)
	}
	decoded, err := base64.StdEncoding.Decode(k.KeyID, trust.Inner)
	if err != nil {
		// If we run into an error, explicitly clear the KeyID, just in case.
		k.KeyID = nil
		return err
	}
	if decoded < len(k.KeyID) {
		k.KeyID = k.KeyID[:decoded]
	}
	return nil
}

// OwnedKeys is a collection of keys that are owned by a particular user.
type OwnedKeys struct {
	Owner jid.JID
	Keys  []Key
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (ok OwnedKeys) TokenReader() xml.TokenReader {
	var keys []xml.TokenReader
	for _, reader := range ok.Keys {
		keys = append(keys, reader.TokenReader())
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(keys...),
		xml.StartElement{
			Name: xml.Name{Local: "key-owner"},
			Attr: []xml.Attr{{Name: xml.Name{Local: "jid"}, Value: ok.Owner.String()}},
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (ok OwnedKeys) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, ok.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (ok OwnedKeys) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := ok.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (ok *OwnedKeys) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	decoded := struct {
		XMLName xml.Name `xml:"key-owner"`
		Addr    jid.JID  `xml:"jid,attr"`
		// Use ",any" and don't check for trust/distrust for two reason:
		// 1. this lets us keep things in order
		// 2. if arbitrary other XML gets mixed in we'll catch it and an error will
		// result when we try to decode it, which will protect us from security
		// issues down the road (I hope).
		Keys []Key `xml:",any"`
	}{}
	err := d.DecodeElement(&decoded, &start)
	if err != nil {
		return err
	}
	ok.Owner = decoded.Addr
	// TODO: should we de-dup owners or leave it as the XML has it, even if
	// someone sends invalid XML?
	ok.Keys = decoded.Keys

	return nil
}

// TrustMessage contains a selection of key owners for a specific encryption
// scheme.
// Each key owner may have multiple keys that are either trusted or explicitly
// distrusted.
type TrustMessage struct {
	Usage      string
	Encryption string
	Keys       []OwnedKeys
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (tm TrustMessage) TokenReader() xml.TokenReader {
	keys := make([]xml.TokenReader, 0, len(tm.Keys))
	for _, v := range tm.Keys {
		keys = append(keys, v.TokenReader())
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(keys...),
		xml.StartElement{
			Name: xml.Name{Space: NSTrust, Local: "trust-message"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "usage"}, Value: tm.Usage},
				{Name: xml.Name{Local: "encryption"}, Value: tm.Encryption},
			},
		},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (tm TrustMessage) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, tm.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (tm TrustMessage) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := tm.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (tm *TrustMessage) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	decoded := struct {
		Usage      string      `xml:"usage,attr"`
		Encryption string      `xml:"encryption,attr"`
		Keys       []OwnedKeys `xml:"key-owner"`
	}{}
	err := d.DecodeElement(&decoded, &start)
	if err != nil {
		return err
	}
	tm.Usage = decoded.Usage
	tm.Encryption = decoded.Encryption
	tm.Keys = decoded.Keys

	return nil
}
