// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"sort"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// InfoQuery is the payload of a query for a node's identities and features.
type InfoQuery struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/disco#info query"`
	Node    string   `xml:"node,attr,omitempty"`
}

func (q InfoQuery) wrap(r xml.TokenReader) xml.TokenReader {
	start := xml.StartElement{Name: xml.Name{Space: NSInfo, Local: "query"}}
	if q.Node != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "node"}, Value: q.Node})
	}
	return xmlstream.Wrap(r, start)
}

// TokenReader implements xmlstream.Marshaler.
func (q InfoQuery) TokenReader() xml.TokenReader {
	return q.wrap(nil)
}

// WriteXML implements xmlstream.WriterTo.
func (q InfoQuery) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, q.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (q InfoQuery) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := q.WriteXML(e)
	return err
}

// Info is a response to a disco info query.
type Info struct {
	InfoQuery
	Identity []info.Identity `xml:"identity"`
	Features []info.Feature  `xml:"feature"`
	Form     []form.Data     `xml:"jabber:x:data x,omitempty"`
}

// TokenReader implements xmlstream.Marshaler.
func (i Info) TokenReader() xml.TokenReader {
	var payloads []xml.TokenReader
	for _, f := range i.Features {
		payloads = append(payloads, f.TokenReader())
	}
	for _, ident := range i.Identity {
		payloads = append(payloads, ident.TokenReader())
	}
	return i.InfoQuery.wrap(xmlstream.MultiReader(payloads...))
}

// WriteXML implements xmlstream.WriterTo.
func (i Info) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, i.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (i Info) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	_, err := i.WriteXML(e)
	return err
}

// Hash generates the entity capabilities verification string.
// Its output is suitable for use as a cache key.
func (i Info) Hash(h hash.Hash) string {
	return string(i.AppendHash(nil, h))
}

// AppendHash is like Hash except that it appends the output string to the
// provided byte slice.
func (i Info) AppendHash(dst []byte, h hash.Hash) []byte {
	// Hash identities
	// TODO: does this match RFC 4790 § 9.3?
	sort.Slice(i.Identity, func(a, b int) bool {
		identI, identJ := i.Identity[a], i.Identity[b]
		if identI.Category != identJ.Category {
			return identI.Category < identJ.Category
		}
		if identI.Type != identJ.Type {
			return identI.Type < identJ.Type
		}
		if identI.Lang != identJ.Lang {
			return identI.Lang < identJ.Lang
		}
		return false
	})
	for _, ident := range i.Identity {
		/* #nosec */
		fmt.Fprintf(h, "%s/%s/%s/%s<", ident.Category, ident.Type, ident.Lang, ident.Name)
	}

	// Hash features
	sort.Slice(i.Features, func(a, b int) bool {
		return i.Features[a].Var < i.Features[b].Var
	})
	for _, f := range i.Features {
		/* #nosec */
		io.WriteString(h, f.Var)
		/* #nosec */
		io.WriteString(h, "<")
	}

	// Hash forms
	for _, infoForm := range i.Form {
		var formType string
		fields := make([]string, 0, infoForm.Len()-1)
		infoForm.ForFields(func(f form.FieldData) {
			if f.Var == "FORM_TYPE" {
				formType, _ = infoForm.GetString("FORM_TYPE")
				return
			}
			fields = append(fields, f.Var)
		})
		sort.Strings(fields)
		/* #nosec */
		io.WriteString(h, formType)
		/* #nosec */
		io.WriteString(h, "<")
		for _, f := range fields {
			/* #nosec */
			io.WriteString(h, f)
			/* #nosec */
			io.WriteString(h, "<")
			vals, _ := infoForm.Raw(f)
			sort.Strings(vals)
			for _, val := range vals {
				/* #nosec */
				io.WriteString(h, val)
				/* #nosec */
				io.WriteString(h, "<")
			}
		}
	}

	dst = h.Sum(dst)
	out := make([]byte, base64.StdEncoding.EncodedLen(len(dst)))
	base64.StdEncoding.Encode(out, dst)
	return out
}

// GetInfo discovers a set of features and identities associated with a JID and
// optional node.
// An empty Node means to query the root items for the JID.
// It blocks until a response is received.
func GetInfo(ctx context.Context, node string, to jid.JID, s *xmpp.Session) (Info, error) {
	return GetInfoIQ(ctx, node, stanza.IQ{To: to}, s)
}

// GetInfoIQ is like GetInfo but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
func GetInfoIQ(ctx context.Context, node string, iq stanza.IQ, s *xmpp.Session) (Info, error) {
	if iq.Type != stanza.GetIQ {
		iq.Type = stanza.GetIQ
	}
	query := InfoQuery{
		Node: node,
	}
	var info Info
	err := s.UnmarshalIQElement(ctx, query.TokenReader(), iq, &info)
	return info, err
}
