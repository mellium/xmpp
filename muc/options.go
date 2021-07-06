// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package muc

import (
	"encoding/xml"
	"math"
	"strconv"
	"time"

	"mellium.im/xmlstream"
)

type historyConfig struct {
	maxStanzas *uint64
	maxChars   *uint64
	seconds    *uint64
	since      *string
}

func optionalString(s string, name xml.Name) xml.TokenReader {
	if s == "" {
		return nil
	}

	return xmlstream.Wrap(
		xmlstream.Token(xml.CharData(s)),
		xml.StartElement{Name: name},
	)
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (h historyConfig) TokenReader() xml.TokenReader {
	if h.maxStanzas == nil && h.maxChars == nil && h.seconds == nil && h.since == nil {
		return nil
	}

	attrs := make([]xml.Attr, 0, 4)
	if h.maxStanzas != nil {
		attrs = append(attrs, xml.Attr{
			Name:  xml.Name{Local: "maxstanzas"},
			Value: strconv.FormatUint(*h.maxStanzas, 10),
		})
	}
	if h.maxChars != nil {
		attrs = append(attrs, xml.Attr{
			Name:  xml.Name{Local: "maxchars"},
			Value: strconv.FormatUint(*h.maxChars, 10),
		})
	}
	if h.seconds != nil {
		attrs = append(attrs, xml.Attr{
			Name:  xml.Name{Local: "seconds"},
			Value: strconv.FormatUint(*h.seconds, 10),
		})
	}
	if h.since != nil {
		attrs = append(attrs, xml.Attr{
			Name:  xml.Name{Local: "since"},
			Value: *h.since,
		})
	}

	return xmlstream.Wrap(
		nil,
		xml.StartElement{Name: xml.Name{Local: "history"}, Attr: attrs},
	)
}

type config struct {
	history  historyConfig
	password string
	newNick  string
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (c config) TokenReader() xml.TokenReader {
	return xmlstream.Wrap(
		xmlstream.MultiReader(
			c.history.TokenReader(),
			optionalString(c.password, xml.Name{Local: "password"}),
		),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "x"}},
	)
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (c config) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, c.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (c config) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := c.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (c *config) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	iter := xmlstream.NewIter(d)
	for iter.Next() {
		start, r := iter.Current()
		switch start.Name.Local {
		case "history":
			for _, attr := range start.Attr {
				switch attr.Name.Local {
				case "maxchars":
					v, err := strconv.ParseUint(attr.Value, 10, 64)
					if err != nil {
						return err
					}
					c.history.maxChars = &v
				case "maxstanzas":
					v, err := strconv.ParseUint(attr.Value, 10, 64)
					if err != nil {
						return err
					}
					c.history.maxStanzas = &v
				case "seconds":
					v, err := strconv.ParseUint(attr.Value, 10, 64)
					if err != nil {
						return err
					}
					c.history.seconds = &v
				case "since":
					c.history.since = &attr.Value
				}
			}
		case "password":
			tok, err := r.Token()
			if err != nil {
				return nil
			}
			cdata, ok := tok.(xml.CharData)
			if ok {
				c.password = string(cdata)
			}
		}
	}
	return iter.Err()
}

// Option is used to configure joining a channel.
type Option func(*config)

// MaxHistory configures the maximum number of messages that will be sent to the
// client when joining the room.
func MaxHistory(messages uint64) Option {
	return func(c *config) {
		c.history.maxStanzas = &messages
	}
}

// MaxBytes configures the maximum number of bytes of XML that will be sent to
// the client when joining the room.
func MaxBytes(b uint64) Option {
	return func(c *config) {
		c.history.maxChars = &b
	}
}

// Duration configures the room to send history received within a window of
// time.
func Duration(d time.Duration) Option {
	return func(c *config) {
		s := uint64(math.Abs(math.Round(d.Seconds())))
		c.history.seconds = &s
	}
}

// Since configures the room to send history received since the provided time.
func Since(t time.Time) Option {
	return func(c *config) {
		s := t.UTC().Format(time.RFC3339Nano)
		c.history.since = &s
	}
}

// Password is used to join password protected rooms.
func Password(p string) Option {
	return func(c *config) {
		c.password = p
	}
}

// Nick overrides the resourcepart of the JID and sets a different nickname in
// the room.
//
// This is mostly useful if you want to change the nickname when re-joining a
// room (when the JID is already known and is not provided to the method) or
// when the room JID is known and you want to let this package handle any errors
// encountered when appending the nickname and reduce boilerplate in your own
// code.
func Nick(n string) Option {
	return func(c *config) {
		c.newNick = n
	}
}
