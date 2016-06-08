// +build ignore

// The echobot command is a pretend XMPP bot written using the (as yet
// non-existant) mellium.im/xmpp package. It is to experiment with and prove out
// the API for mellium.im/xmpp and won't actually compile (yet).
package main

import (
	"context"
	"log"

	"example.net/sasl" // This package was made up and doesn't exist yet.
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
)

var laddr = jid.MustParse("echo@mellium.im")

func main() {

	// TODO: We could also have an xmpp.NewConn(c net.Conn) function that wraps an
	// existing connection in an xmpp.Conn. This way we could implement
	// xmpp-over-random-transport without really thinking about it (websockets and
	// BOSH would probably be handled at a higher level since they require some
	// modifications to the underlying XML stream).

	// Supports all the normal net.Dial networks (tcp, tcp4, udp, etc.), but also
	// knows what to do if you use "websockets" or "bosh" and how to discover the
	// endpoint if it's different from the specified address.
	c, err := xmpp.DialClient(context.Background(), "tcp6", laddr)
	if err != nil {
		log.Fatal("Failed to dial upstream XMPP server")
	}

	// xmpp.Conn implements net.Conn, so you can fmt.Fprintf(c, "<somexml/>") if
	// you really want too (but you probably don't).

	// TODO: What does Auth look like?
	c.Auth(sasl.ScramSHA1Plus(c.TLSConfig), sasl.DigestMD5)

	// TODO: What does a roster request look like? Is it worth having a special
	// function, or should the user construct a request?
	// For the echobot example we may not care.

	// Send initial presence, but for an echobot we may not care.
	// Send does not wait for a response. The sent stanza is returned.
	// stanza := c.Send(xmpp.Presence{})

	// If we wanted to send an IQ instead, we might use the AwaitResponse method
	// which waits for a resonse with the same ID as the given stanza and then
	// returns it (or returns an error). This should probably only be used for IQs
	// and can be used to implement retries. If the response is an error, it is
	// returned as resp, not err. If stanza does not have an ID assigned already,
	// panic.
	//
	// TODO: Should this be limited to IQs, or can it handle any stanza (so that
	// it can be reused for protocols that may respond to broadcast messages).
	// resp, err := c.AwaitResponse(ctx, c.Send(xmpp.IQ{}))

	c.HandleIQ(xml.Name{"urn:xmpp:ping", "ping"})

	// Or, if we wanted to handle all IQs ourself:
	// c.HandleIQ(xml.Name{})

	// TODO: Should s.Receive be like the websocket codec implementation, or
	// return a channel?
	for stanza := range <-c.Receive() {
		switch ts := stanza.(type) {
		case xmpp.Message:
			m := ts.Copy()
			m.to, m.from = m.from, m.to

			// Send should be variadic and take multiple addresses to send the message
			// too.  If no jids are specified, it should use the one in the message
			// already, if there's not one in the message, it should send to the
			// server.
			//
			// TODO: Or should it panic if there's nowhere to send the message?
			s.Send(m)
		case xmpp.IQ:
			// Reply to the ping with a result (we know it's a ping because that's all
			// we're handling right now).
			c.Reply(ts, xmpp.IQ{Type: ResultIQ})
		default:
			// Throw it away!
		}
	}
}
