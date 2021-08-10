// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package history

import (
	"encoding/xml"
	"time"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/paging"
)

// Query is a request to the archive for data.
// An empty query indicates all messages should be fetched without a filter and
// with a random ID.
type Query struct {
	// Query parameters
	ID string

	// Filters
	With     jid.JID
	Start    time.Time
	End      time.Time
	BeforeID string
	AfterID  string
	IDs      []string

	Limit paging.RequestNext
}

const (
	fieldWith   = "with"
	fieldStart  = "start"
	fieldEnd    = "end"
	fieldAfter  = "after-id"
	fieldBefore = "before-id"
	fieldIDs    = "ids"
)

// TokenReader implements xmlstream.Marshaler.
func (f Query) TokenReader() xml.TokenReader {
	dataForm := form.New(
		form.Hidden("FORM_TYPE", form.Value(NS)),
		form.JID(fieldWith),
		form.Text(fieldStart),
		form.Text(fieldEnd),
		form.Text(fieldAfter),
		form.Text(fieldBefore),
		form.ListMulti(fieldIDs),
	)
	if !f.With.Equal(jid.JID{}) {
		/* #nosec */
		dataForm.Set(fieldWith, f.With.String())
	}
	if !f.Start.IsZero() {
		/* #nosec */
		dataForm.Set(fieldStart, f.Start.Format(time.RFC3339))
	}
	if !f.End.IsZero() {
		/* #nosec */
		dataForm.Set(fieldStart, f.End.Format(time.RFC3339))
	}
	if f.AfterID != "" {
		/* #nosec */
		dataForm.Set(fieldAfter, f.AfterID)
	}
	if f.BeforeID != "" {
		/* #nosec */
		dataForm.Set(fieldBefore, f.BeforeID)
	}
	if len(f.IDs) > 0 {
		/* #nosec */
		dataForm.Set(fieldIDs, f.IDs)
	}
	filter, _ := dataForm.Submit()
	return xmlstream.Wrap(
		filter,
		xml.StartElement{
			Name: xml.Name{Space: NS, Local: "query"},
			Attr: []xml.Attr{{Name: xml.Name{Local: "queryid"}, Value: f.ID}},
		},
	)
}

// WriteXML implements xmlstream.WriterTo.
func (f *Query) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, f.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (f *Query) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := f.WriteXML(e)
	return err
}
