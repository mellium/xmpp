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

// NS is the namespace used by this package, provided as a convenience.
const NS = `urn:xmpp:blocklist`

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
