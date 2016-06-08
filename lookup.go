// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/jid"
)

const (
	wsPrefix    = "_xmpp-client-websocket="
	boshPrefix  = "_xmpp-client-xbosh="
	wsRel       = "urn:xmpp:alt-connections:websocket"
	boshRel     = "urn:xmpp:alt-connections:xbosh"
	hostMetaXML = "/.well-known/host-meta"
)

var (
	xrdName = xml.Name{
		Space: "http://docs.oasis-open.org/ns/xri/xrd-1.0",
		Local: "XRD",
	}
)

// LookupWebsocket discovers websocket endpoints that are valid for the given
// address using DNS TXT records and Web Host Metadata as described in XEP-0156.
// If client is nil, only DNS is queried.
func LookupWebsocket(ctx context.Context, client *http.Client, addr *jid.JID) (urls []string, err error) {
	return lookupEndpoint(ctx, client, addr, "ws")
}

// LookupBOSH discovers BOSH endpoints that are valid for the given address
// using DNS TXT records and Web Host Metadata as described in XEP-0156. If
// client is nil, only DNS is queried.
func LookupBOSH(ctx context.Context, client *http.Client, addr *jid.JID) (urls []string, err error) {
	return lookupEndpoint(ctx, client, addr, "bosh")
}

func validateConnTypeOrPanic(conntype string) {

	if conntype != "ws" && conntype != "bosh" {
		panic("xmpp.lookupEndpoint: Invalid conntype specified")
	}
}

func lookupEndpoint(ctx context.Context, client *http.Client, addr *jid.JID, conntype string) (urls []string, err error) {
	validateConnTypeOrPanic(conntype)

	// TODO: Should these even be fetched concurrently?
	var (
		u  []string
		e  error
		wg sync.WaitGroup

		name = addr.Domainpart()
	)

	wg.Add(1)
	go func() {
		urls, err = lookupDNS(ctx, name, conntype)
		wg.Done()
	}()
	if client != nil {
		wg.Add(1)
		go func() {
			u, e = lookupHostMeta(ctx, client, name, conntype)
			wg.Done()
		}()
	}
	wg.Wait()

	switch {
	case err == nil && len(urls) > 0:
		return urls, err
	case e == nil && len(u) > 0:
		return u, e
	case err != nil:
		return urls, err
	case e != nil:
		return u, e
	}

	return urls, err
}

// TODO(ssw): Rely on the OS DNS cache, or cache lookups ourselves?

func lookupDNS(ctx context.Context, name, conntype string) (urls []string, err error) {
	validateConnTypeOrPanic(conntype)

	txts, err := net.LookupTXT(name)
	if err != nil {
		return urls, err
	}

	var s string
	for _, txt := range txts {
		if _, ok := <-ctx.Done(); ok {
			return urls, ctx.Err()
		}
		switch conntype {
		case "ws":
			if s = strings.TrimPrefix(txt, wsPrefix); s != txt {
				urls = append(urls, s)
			}
		case "bosh":
			if s = strings.TrimPrefix(txt, boshPrefix); s != txt {
				urls = append(urls, s)
			}
		}
	}

	return urls, err
}

// TODO(ssw): Memoize the following functions?

func lookupHostMeta(ctx context.Context, client *http.Client, name, conntype string) (urls []string, err error) {
	validateConnTypeOrPanic(conntype)

	url, err := url.Parse(name)
	if err != nil {
		return urls, err
	}
	url.Path = ""

	xrd, err := getHostMetaXML(ctx, client, url.String())
	if err != nil {
		return urls, err
	}

	for _, link := range xrd.Links {
		switch conntype {
		case "ws":
			if link.Rel == wsRel {
				urls = append(urls, link.Href)
			}
		case "bosh":
			if link.Rel == boshRel {
				urls = append(urls, link.Href)
			}
		}
	}
	return urls, err
}

func getHostMetaXML(
	ctx context.Context, client *http.Client, name string) (xrd internal.XRD, err error) {
	resp, err := ctxhttp.Get(ctx, client, path.Join(name, hostMetaXML))
	if err != nil {
		return xrd, err
	}
	d := xml.NewDecoder(resp.Body)

	t, err := d.Token()
	for {
		select {
		case <-ctx.Done():
			return xrd, ctx.Err()
		default:
			if se, ok := t.(xml.StartElement); ok && se.Name == xrdName {
				if err = d.DecodeElement(&xrd, &se); err != nil {
					return xrd, err
				}
				break
			}
		}
	}
}
