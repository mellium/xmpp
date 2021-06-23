// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package muc

import (
	"context"
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

var directName = xml.Name{Space: NSConf, Local: "x"}

type inviteHandler struct {
	F func(Invitation)
}

func (h inviteHandler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	d := xml.NewTokenDecoder(t)
	// Pop the <message> token.
	_, err := d.Token()
	if err != nil {
		return err
	}
	var x Invitation
	err = d.Decode(&x)
	if err != nil {
		return err
	}

	if h.F != nil {
		h.F(x)
		return nil
	}
	return nil
}

// HandleInvite returns an option that registers a handler for direct MUC
// invitations.
// To handle mediated invitations register a client handler using HandleClient.
func HandleInvite(f func(Invitation)) mux.Option {
	return func(m *mux.ServeMux) {
		msgName := xml.Name{Space: NSConf, Local: "x"}

		mux.Message(stanza.NormalMessage, msgName, inviteHandler{
			F: f,
		})(m)
	}
}

// Invitation is a mediated or direct MUC invitation.
// When the XML is marshaled or unmarshaled the namespace determines whether the
// invitation was direct or mediated.
// The default is mediated.
type Invitation struct {
	XMLName  xml.Name
	Continue bool
	JID      jid.JID
	Password string
	Reason   string
	Thread   string
}

// MarshalDirect returns the invitation as a direct MUC invitation (sent
// directly to the invitee).
func (i Invitation) MarshalDirect() xml.TokenReader {
	attr := []xml.Attr{{
		Name:  xml.Name{Local: "jid"},
		Value: i.JID.String(),
	}}
	if i.Continue {
		attr = append(attr, xml.Attr{
			Name:  xml.Name{Local: "continue"},
			Value: "true",
		})
		if i.Thread != "" {
			attr = append(attr, xml.Attr{
				Name:  xml.Name{Local: "thread"},
				Value: i.Thread,
			})
		}
	}
	if i.Password != "" {
		attr = append(attr, xml.Attr{
			Name:  xml.Name{Local: "password"},
			Value: i.Password,
		})
	}
	if i.Reason != "" {
		attr = append(attr, xml.Attr{
			Name:  xml.Name{Local: "reason"},
			Value: i.Reason,
		})
	}
	return xmlstream.Wrap(
		nil,
		xml.StartElement{Name: i.XMLName, Attr: attr},
	)
}

// MarshalMediated returns the invitation as a mediated MUC invitation (sent
// to the room and then forwarded to the invitee).
func (i Invitation) MarshalMediated() xml.TokenReader {
	var reasonEl, passEl, continueEl xml.TokenReader
	if i.Reason != "" {
		reasonEl = xmlstream.Wrap(
			xmlstream.Token(xml.CharData(i.Reason)),
			xml.StartElement{Name: xml.Name{Local: "reason"}},
		)
	}
	if i.Password != "" {
		passEl = xmlstream.Wrap(
			xmlstream.Token(xml.CharData(i.Password)),
			xml.StartElement{Name: xml.Name{Local: "password"}},
		)
	}
	if i.Continue {
		var attr []xml.Attr
		if i.Thread != "" {
			attr = []xml.Attr{{
				Name:  xml.Name{Local: "thread"},
				Value: i.Thread,
			}}
		}
		continueEl = xmlstream.Wrap(
			nil,
			xml.StartElement{Name: xml.Name{Local: "continue"}, Attr: attr},
		)
	}
	return xmlstream.Wrap(
		xmlstream.MultiReader(
			xmlstream.Wrap(
				xmlstream.MultiReader(
					reasonEl,
					continueEl,
				),
				xml.StartElement{
					Name: xml.Name{Local: "invite"},
					Attr: []xml.Attr{{Name: xml.Name{Local: "to"}, Value: i.JID.String()}},
				},
			),
			passEl,
		),
		xml.StartElement{Name: xml.Name{Space: NSUser, Local: "x"}},
	)
}

// TokenReader satisfies the xmlstream.Marshaler interface.
//
// It calls either MarshalDirect or MarshalMediated depending on the invitations
// XMLName field.
func (i Invitation) TokenReader() xml.TokenReader {
	// Direct invite
	if i.XMLName == directName {
		return i.MarshalDirect()
	}

	// Mediated invite
	return i.MarshalMediated()
}

// WriteXML satisfies the xmlstream.WriterTo interface.
// It is like MarshalXML except it writes tokens to w.
func (i Invitation) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, i.TokenReader())
}

// MarshalXML implements xml.Marshaler.
func (i Invitation) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := i.WriteXML(e)
	return err
}

// UnmarshalXML implements xml.Unmarshaler.
func (i *Invitation) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if start.Name == directName {
		s := struct {
			XMLName  xml.Name `xml:"jabber:x:conference x"`
			Continue bool     `xml:"continue,attr"`
			JID      jid.JID  `xml:"jid,attr"`
			Pass     string   `xml:"password,attr"`
			Reason   string   `xml:"reason,attr"`
			Thread   string   `xml:"thread,attr"`
		}{}
		err := d.DecodeElement(&s, &start)
		if err != nil {
			return err
		}
		i.XMLName = s.XMLName
		i.Continue = s.Continue
		i.JID = s.JID
		i.Password = s.Pass
		i.Reason = s.Reason
		i.Thread = s.Thread
		return nil
	}

	s := struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/muc#user x"`
		Invite  struct {
			To       jid.JID `xml:"to,attr"`
			Reason   string  `xml:"reason"`
			Continue struct {
				XMLName xml.Name
				Thread  string `xml:"thread,attr"`
			} `xml:"continue"`
		} `xml:"invite"`
		Pass string `xml:"password"`
	}{}
	err := d.DecodeElement(&s, &start)
	if err != nil {
		return err
	}
	i.XMLName = s.XMLName
	i.Continue = s.Invite.Continue.XMLName.Local != ""
	i.JID = s.Invite.To
	i.Password = s.Pass
	i.Reason = s.Invite.Reason
	i.Thread = s.Invite.Continue.Thread
	return nil
}

// Invite sends a direct MUC invitation using the provided session.
// This is useful when a mediated invitation (one sent through the channel using
// the Invite method) is being blocked by a user that does not allow contact
// from unrecognized JIDs.
// Changing the XMLName field of the invite has no effect.
func Invite(ctx context.Context, to jid.JID, invite Invitation, s *xmpp.Session) error {
	invite.XMLName = directName
	return s.Send(ctx, stanza.Message{
		To:   to,
		Type: stanza.NormalMessage,
	}.Wrap(invite.MarshalDirect()))
}
