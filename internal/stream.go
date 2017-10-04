// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package internal

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stream"
)

const (
	XMLHeader = `<?xml version="1.0" encoding="UTF-8"?>`
)

type StreamInfo struct {
	to      *jid.JID
	from    *jid.JID
	id      string
	version Version
	xmlns   string
	lang    string
}

// This MUST only return stream errors.
// TODO: Is the above true? Just make it return a StreamError?
func streamFromStartElement(s xml.StartElement) (StreamInfo, error) {
	streamData := StreamInfo{}
	for _, attr := range s.Attr {
		switch attr.Name {
		case xml.Name{Space: "", Local: "to"}:
			streamData.to = &jid.JID{}
			if err := streamData.to.UnmarshalXMLAttr(attr); err != nil {
				return streamData, stream.ImproperAddressing
			}
		case xml.Name{Space: "", Local: "from"}:
			streamData.from = &jid.JID{}
			if err := streamData.from.UnmarshalXMLAttr(attr); err != nil {
				return streamData, stream.ImproperAddressing
			}
		case xml.Name{Space: "", Local: "id"}:
			streamData.id = attr.Value
		case xml.Name{Space: "", Local: "version"}:
			(&streamData.version).UnmarshalXMLAttr(attr)
		case xml.Name{Space: "", Local: "xmlns"}:
			if attr.Value != "jabber:client" && attr.Value != "jabber:server" {
				return streamData, stream.InvalidNamespace
			}
			streamData.xmlns = attr.Value
		case xml.Name{Space: "xmlns", Local: "stream"}:
			if attr.Value != ns.Stream {
				return streamData, stream.InvalidNamespace
			}
		case xml.Name{Space: "xml", Local: "lang"}:
			streamData.lang = attr.Value
		}
	}
	return streamData, nil
}

// Sends a new XML header followed by a stream start element on the given
// io.Writer. We don't use an xml.Encoder both because Go's standard library xml
// package really doesn't like the namespaced stream:stream attribute and
// because we can guarantee well-formedness of the XML with a print in this case
// and printing is much faster than encoding. Afterwards, clear the
// StreamRestartRequired bit and set the output stream information.
func SendNewStream(rw io.ReadWriter, s2s bool, version Version, lang string, location, origin, id string) (StreamInfo, error) {
	streamData := StreamInfo{}
	switch s2s {
	case true:
		streamData.xmlns = ns.Server
	case false:
		streamData.xmlns = ns.Client
	}

	streamData.id = id
	if id == "" {
		id = " "
	} else {
		id = ` id='` + id + `' `
	}

	_, err := fmt.Fprintf(rw,
		XMLHeader+`<stream:stream%sto='%s' from='%s' version='%s' xml:lang='`,
		id,
		location,
		origin,
		version,
	)
	if err != nil {
		return streamData, err
	}

	err = xml.EscapeText(rw, []byte(lang))
	if err != nil {
		return streamData, err
	}

	_, err = fmt.Fprintf(rw, `' xmlns='%s' xmlns:stream='http://etherx.jabber.org/streams'>`,
		streamData.xmlns,
	)
	if err != nil {
		return streamData, err
	}

	return streamData, nil
}

func ExpectNewStream(ctx context.Context, d xml.TokenReader, recv bool) (streamData StreamInfo, err error) {
	var foundHeader bool

	for {
		select {
		case <-ctx.Done():
			return streamData, ctx.Err()
		default:
		}
		t, err := d.Token()
		if err != nil {
			return streamData, err
		}
		switch tok := t.(type) {
		case xml.StartElement:
			switch {
			case tok.Name.Local == "error" && tok.Name.Space == ns.Stream:
				se := stream.Error{}
				if err := xml.NewTokenDecoder(d).DecodeElement(&se, &tok); err != nil {
					return streamData, err
				}
				return streamData, se
			case tok.Name.Local != "stream":
				return streamData, stream.BadFormat
			case tok.Name.Space != ns.Stream:
				return streamData, stream.InvalidNamespace
			}

			streamData, err = streamFromStartElement(tok)
			switch {
			case err != nil:
				return streamData, err
			case streamData.version != DefaultVersion:
				return streamData, stream.UnsupportedVersion
			}

			if !recv && streamData.id == "" {
				// if we are the initiating entity and there is no stream ID…
				return streamData, stream.BadFormat
			}
			return streamData, nil
		case xml.ProcInst:
			// TODO: If version or encoding are declared, validate XML 1.0 and UTF-8
			if !foundHeader && tok.Target == "xml" {
				foundHeader = true
				continue
			}
			return streamData, stream.RestrictedXML
		case xml.EndElement:
			return streamData, stream.NotWellFormed
		default:
			return streamData, stream.RestrictedXML
		}
	}
}
