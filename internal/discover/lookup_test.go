// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package discover

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"mellium.im/xmpp/jid"
)

var (
	boshLink = Link{Rel: "urn:xmpp:alt-connections:xbosh", Href: "https://web.example.com:5280/bosh"}
	wsLink   = Link{Rel: "urn:xmpp:alt-connections:websocket", Href: "wss://web.example.com:443/ws"}
)

func TestUnmarshalWellKnownXML(t *testing.T) {
	hostMeta := []byte(`<XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>
  <Link rel="urn:xmpp:alt-connections:xbosh"
        href="https://web.example.com:5280/bosh" />
  <Link rel="urn:xmpp:alt-connections:websocket"
        href="wss://web.example.com:443/ws" />
</XRD>`)
	var xrd XRD
	if err := xml.Unmarshal(hostMeta, &xrd); err != nil {
		t.Error(err)
	}
	switch {
	case len(xrd.Links) != 2:
		t.Errorf("Expected 2 links in xrd unmarshal output, but found %d", len(xrd.Links))
	case xrd.Links[0] != boshLink:
		t.Errorf("Expected %v, but got %v", boshLink, xrd.Links[0])
	case xrd.Links[1] != wsLink:
		t.Errorf("Expected %v, but got %v", wsLink, xrd.Links[1])
	}
}

// If an invalid connection type is looked up, we should panic.
func TestLookupHostMetaPanicsOnInvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("lookupHostMeta should panic if an invalid conntype is specified.")
		}
	}()
	lookupHostMeta(context.Background(), nil, "name", "wssorbashorsomething")
}

// portSchemeRoundTripper is an http.RoundTripper that wraps an existing round
// tripper and changes the port and scheme for all outgoing requests.
// If the scheme is not initially https, RoundTrip returns an error.
type portSchemeRoundTripper struct {
	port string
	rt   http.RoundTripper
}

func (pr portSchemeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != "https" {
		return nil, fmt.Errorf("wrong scheme found: want=https, got=%s", r.URL.Scheme)
	}
	r.URL.Scheme = "http"
	r.URL.Host = net.JoinHostPort(r.URL.Host, pr.port)
	return pr.rt.RoundTrip(r)
}

func TestLookupXRD(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("error listening for TCP connections: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	fakeWSAddrs := []string{
		"one",
		"two",
		"three",
	}
	fakeBOSHAddrs := []string{
		"four",
		"five",
	}
	s := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const want = "/.well-known/host-meta"
			if r.URL.Path != want {
				http.Error(w, fmt.Sprintf("wrong path: want=%v, got=%v", want, r.URL.Path), http.StatusNotFound)
				return
			}
			_, err := fmt.Fprint(w, `<?xml version='1.0' encoding='utf-8'?>
			<XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>`)
			if err != nil {
				panic(err)
			}
			for _, addr := range fakeWSAddrs {
				fmt.Fprintf(w, `<Link rel="urn:xmpp:alt-connections:websocket" href="%s" />`, addr)
			}
			for _, addr := range fakeBOSHAddrs {
				fmt.Fprintf(w, `<Link rel="urn:xmpp:alt-connections:xbosh" href="%s" />`, addr)
			}
			_, err = fmt.Fprint(w, `</XRD>`)
			if err != nil {
				panic(err)
			}
		}),
		ReadTimeout:    time.Second,
		WriteTimeout:   time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	defer func() {
		err := s.Close()
		if err != nil {
			t.Logf("error closing HTTP server: %v", err)
		}
	}()
	go func() {
		err := s.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			t.Logf("error serving: %v", err)
		}
	}()

	j := jid.MustParse("me@localhost")
	c := &http.Client{
		Transport: portSchemeRoundTripper{
			port: strconv.Itoa(port),
			rt:   http.DefaultTransport,
		},
	}

	t.Run("WebSocket", func(t *testing.T) {
		addrs, err := LookupWebSocket(context.Background(), c, j)
		if err != nil {
			t.Fatalf("error looking up websocket: %v", err)
		}
		if !reflect.DeepEqual(addrs, fakeWSAddrs) {
			t.Fatalf("got wrong addresses: want=%v, got=%v", fakeWSAddrs, addrs)
		}
	})
	t.Run("BOSH", func(t *testing.T) {
		addrs, err := LookupBOSH(context.Background(), c, j)
		if err != nil {
			t.Fatalf("error looking up websocket: %v", err)
		}
		if !reflect.DeepEqual(addrs, fakeBOSHAddrs) {
			t.Fatalf("got wrong addresses: want=%v, got=%v", fakeBOSHAddrs, addrs)
		}
	})
}
