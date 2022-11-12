// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package blocklist implements blocking and unblocking of contacts.
package blocklist // import "mellium.im/xmpp/blocklist"

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// Various namespaces used by this package, provided as a convenience.
const (
	NS          = `urn:xmpp:blocking`
	NSReporting = `urn:xmpp:reporting:1`
)

// Match checks j1 aginst a JID in the blocklist (j2) and returns true if they
// are a match.
//
// The JID matches the blocklist JID if any of the following compare to the
// blocklist JID (falling back in this order):
//
//   - Full JID (user@domain/resource)
//   - Bare JID (user@domain)
//   - Full domain (domain/resource)
//   - Bare domain
func Match(j1, j2 jid.JID) bool {
	return j1.Equal(j2) ||
		j1.Bare().Equal(j2) ||
		jid.NewUnsafe("", j1.Domainpart(), j1.Resourcepart()).JID.Equal(j2) ||
		j1.Domain().Equal(j2)
}

// Iter is an iterator over blocklist JIDs.
type Iter struct {
	iter    *xmlstream.Iter
	current jid.JID
	err     error
}

// Next returns true if there are more items to decode.
func (i *Iter) Next() bool {
	if i.err != nil || !i.iter.Next() {
		return false
	}
	start, _ := i.iter.Current()
	// If we encounter a lone token that doesn't begin with a start element (eg.
	// a comment) skip it. This should never happen with XMPP, but we don't want
	// to panic in case this somehow happens so just skip it.
	if start == nil {
		return i.Next()
	}
	for _, attr := range start.Attr {
		if attr.Name.Local == "jid" {
			i.current, i.err = jid.Parse(attr.Value)
			break
		}
	}
	return true
}

// Err returns the last error encountered by the iterator (if any).
func (i *Iter) Err() error {
	if i.err != nil {
		return i.err
	}

	return i.iter.Err()
}

// JID returns the last blocked JID parsed by the iterator.
func (i *Iter) JID() jid.JID {
	return i.current
}

// Close indicates that we are finished with the given iterator and processing
// the stream may continue.
// Calling it multiple times has no effect.
func (i *Iter) Close() error {
	if i.iter == nil {
		return nil
	}
	return i.iter.Close()
}

// Fetch sends a request to the JID asking for the blocklist.
func Fetch(ctx context.Context, s *xmpp.Session) *Iter {
	return FetchIQ(ctx, stanza.IQ{}, s)
}

// FetchIQ is like Fetch except that it lets you customize the IQ.
// Changing the type of the provided IQ has no effect.
func FetchIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session) *Iter {
	if iq.Type != stanza.GetIQ {
		iq.Type = stanza.GetIQ
	}
	iter, _, err := s.IterIQ(ctx, iq.Wrap(xmlstream.Wrap(nil, xml.StartElement{
		Name: xml.Name{Space: NS, Local: "blocklist"},
	})))
	if err != nil {
		return &Iter{err: err}
	}
	return &Iter{
		iter: iter,
	}
}

// ReportReason is a reason of a report.
type ReportReason string

// The available report reasons are listed below.
const (
	// ReasonSpam is used for reporting a JID that is sending unwanted messages.
	ReasonSpam ReportReason = "urn:xmpp:reporting:spam"

	// ReasonAbuse is used for reporting general abuse.
	ReasonAbuse ReportReason = "urn:xmpp:reporting:abuse"
)

// Item is a block payload.
// It consists of a JID you want to block and optional report fields.
type Item struct {
	JID       jid.JID
	Reason    ReportReason
	StanzaIDs []stanza.ID
	Text      string
}

// MarshalXML satisfies the xml.Marshaler interface.
func (i *Item) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := i.WriteXML(e)
	return err
}

// UnmarshalXML satisfies the xml.Unmarshaler interface.
func (i *Item) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	data := struct {
		JID    jid.JID `xml:"jid,attr"`
		Report struct {
			XMLName   xml.Name     `xml:"urn:xmpp:reporting:1 report"`
			Reason    ReportReason `xml:"reason,attr"`
			StanzaIDs []stanza.ID  `xml:"stanza-id"`
			Text      string       `xml:"text"`
		}
	}{}
	err := d.DecodeElement(&data, &start)
	if err != nil {
		return err
	}
	i.JID = data.JID
	i.Reason = data.Report.Reason
	i.StanzaIDs = data.Report.StanzaIDs
	i.Text = data.Report.Text
	return nil
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (i *Item) TokenReader() xml.TokenReader {
	var report xml.TokenReader
	if i.Reason != "" || len(i.StanzaIDs) > 0 || i.Text != "" {
		var child []xml.TokenReader
		for _, stanzaID := range i.StanzaIDs {
			child = append(child, stanzaID.TokenReader())
		}
		if i.Text != "" {
			child = append(child, xmlstream.Wrap(
				xmlstream.Token(xml.CharData(i.Text)),
				xml.StartElement{
					Name: xml.Name{Local: "text"},
				},
			))
		}
		reason := ReasonSpam
		if i.Reason != "" {
			reason = i.Reason
		}
		report = xmlstream.Wrap(
			xmlstream.MultiReader(child...),
			xml.StartElement{
				Name: xml.Name{Space: NSReporting, Local: "report"},
				Attr: []xml.Attr{{Name: xml.Name{Local: "reason"}, Value: string(reason)}},
			},
		)
	}
	return xmlstream.Wrap(report, xml.StartElement{
		Name: xml.Name{Local: "item"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "jid"}, Value: i.JID.String()}},
	})
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (i *Item) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, i.TokenReader())
}

// Report adds JIDs to the blocklist.
// You can optionally specify a report for each individual JID.
func Report(ctx context.Context, s *xmpp.Session, i ...Item) error {
	return ReportIQ(ctx, stanza.IQ{}, s, i...)
}

// ReportIQ is like Report except that it lets you customize the IQ.
// Changing the type of the provided IQ has no effect.
func ReportIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session, i ...Item) error {
	if iq.Type != stanza.SetIQ {
		iq.Type = stanza.SetIQ
	}
	var items []xml.TokenReader
	for _, item := range i {
		items = append(items, item.TokenReader())
	}
	r, err := s.SendIQ(ctx, iq.Wrap(xmlstream.Wrap(
		xmlstream.MultiReader(items...),
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "block"},
		},
	)))
	if err != nil {
		return err
	}
	return r.Close()
}

// Add adds JIDs to the blocklist.
func Add(ctx context.Context, s *xmpp.Session, j ...jid.JID) error {
	return AddIQ(ctx, stanza.IQ{}, s, j...)
}

// AddIQ is like Add except that it lets you customize the IQ.
// Changing the type of the provided IQ has no effect.
func AddIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session, j ...jid.JID) error {
	return doIQ(ctx, "block", iq, s, j...)
}

// Remove removes JIDs from the blocklist.
// If no JIDs are provided the entire blocklist is cleared.
func Remove(ctx context.Context, s *xmpp.Session, j ...jid.JID) error {
	return RemoveIQ(ctx, stanza.IQ{}, s, j...)
}

// RemoveIQ is like Remove except that it lets you customize the IQ.
// Changing the type of the provided IQ has no effect.
func RemoveIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session, j ...jid.JID) error {
	return doIQ(ctx, "unblock", iq, s, j...)
}

func doIQ(ctx context.Context, local string, iq stanza.IQ, s *xmpp.Session, j ...jid.JID) error {
	if iq.Type != stanza.SetIQ {
		iq.Type = stanza.SetIQ
	}
	var jids []xml.TokenReader
	for _, jj := range j {
		jids = append(jids, xmlstream.Wrap(nil, xml.StartElement{
			Name: xml.Name{Local: "item"},
			Attr: []xml.Attr{{Name: xml.Name{Local: "jid"}, Value: jj.String()}},
		}))
	}
	r, err := s.SendIQ(ctx, iq.Wrap(xmlstream.Wrap(
		xmlstream.MultiReader(jids...),
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: local},
		},
	)))
	if err != nil {
		return err
	}
	return r.Close()
}
