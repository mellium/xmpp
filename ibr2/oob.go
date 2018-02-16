// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibr2

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/oob"
)

// OOB is a challenge that must be completed out of band using a URI provided by
// XEP-0066: Out of Band Data.
// If you are a client, f will be called and passed the parsed OOB data.
// If f returns an error, the client considers the negotiation a failure.
// For servers, the provided data is encoded and sent as part of the challenge.
func OOB(data *oob.Data, f func(*oob.Data) error) Challenge {
	return Challenge{
		Type: oob.NS,
		Send: func(ctx context.Context, w xmlstream.TokenWriter) (err error) {
			if err = writeDataTo(w, data); err != nil {
				return err
			}
			return w.Flush()
		},
		Receive: func(ctx context.Context, server bool, r xml.TokenReader, start *xml.StartElement) error {
			// The server does not receive a reply for this mechanism.
			if server {
				return nil
			}

			oob := &oob.Data{}
			err := xml.NewTokenDecoder(r).Decode(oob)
			if err != nil {
				return err
			}

			return f(oob)
		},
	}
}

var (
	xStartToken = xml.StartElement{
		Name: xml.Name{Space: oob.NS, Local: "x"},
	}
	urlStartToken = xml.StartElement{
		Name: xml.Name{Local: "url"},
	}
	descStartToken = xml.StartElement{
		Name: xml.Name{Local: "desc"},
	}
)

// TODO: move this to the mellium.im/xmpp/oob package as a method on Data?
// Also, possibly add a matching interface in mellium.im/xmlstream.

// writeDataTo encodes d to w.
func writeDataTo(w xmlstream.TokenWriter, d *oob.Data) (err error) {
	if err = w.EncodeToken(xStartToken); err != nil {
		return err
	}
	if err = w.EncodeToken(urlStartToken); err != nil {
		return err
	}
	if err = w.EncodeToken(xml.CharData(d.URL)); err != nil {
		return err
	}
	if err = w.EncodeToken(urlStartToken.End()); err != nil {
		return err
	}
	if err = w.EncodeToken(descStartToken); err != nil {
		return err
	}
	if err = w.EncodeToken(xml.CharData(d.Desc)); err != nil {
		return err
	}
	if err = w.EncodeToken(descStartToken.End()); err != nil {
		return err
	}
	return w.EncodeToken(xStartToken.End())
}
