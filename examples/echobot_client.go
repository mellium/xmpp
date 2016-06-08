// +build ignore

// The echobot command is a pretend XMPP bot written using the (as yet
// non-existant) higher level mellium.im/xmpp/client package. It is to
// experiment with and prove out the API; it won't actually compile (yet).
package main

import (
	"log"

	"mellium.im/xmpp/client"
	"mellium.im/xmpp/jid"
)

var laddr = jid.MustParse("echo@mellium.im")

func main() {
	// New is variadic and takes an arbitrary number of client.Option's
	c, err := client.New(laddr, client.TLSConfig(tls.Config{}))

	// TODO: Most libraries probably let you match on xpath or something so you
	//       can, eg. only handle messages that contain a <thread/> child element.
	//       I don't want that overhead (and there's no built in xpath anyways),
	//       so maybe we need a way to get/trigger the default handler if the
	//       method doesn't meet some sort of validation here within our handler?
	//       Maybe handlers should be a middleware style pattern and we can fall
	//       through to the next handler in the chain, or return and short
	//       circuit?
	c.HandleIQ(xml.Name{"urn:xmpp:ping", "ping"}, func(event client.Event) {
		log.Println("Got ping, sending pong.")
		event.Client.Reply(event.Stanza, xmpp.IQ{Type: ResultIQ})
	})

	if err := c.Connect(ctx); err != nil {
		log.Fatalln("echobot failed to connect: ", err)
	}
	c.GetRoster()
	c.SendPresence()

	// Block until the client logs out (or the connection is severed in some other
	// way). When we called client.New a goroutine was spun up which will keep
	// processing the client connection in the background, so the main thread can
	// happily go to sleep.
	<-c.Done()
}
