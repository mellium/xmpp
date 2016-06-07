// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/json"
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
	wsPrefix     = "_xmpp-client-websocket="
	boshPrefix   = "_xmpp-client-xbosh="
	wsRel        = "urn:xmpp:alt-connections:websocket"
	boshRel      = "urn:xmpp:alt-connections:xbosh"
	hostMetaXML  = "/.well-known/host-meta"
	hostMetaJSON = "/.well-known/host-meta.json"
)

var (
	xrdName = xml.Name{
		Space: "http://docs.oasis-open.org/ns/xri/xrd-1.0",
		Local: "XRD",
	}
)

// LookupWebsocket discovers websocket endpoints that are valid for the given
// address using DNS TXT records and Web Host Metadata as described in XEP-0156:
// Discovering Alternative XMPP Connection Methods. If client is nil, only DNS
// is queried.
func LookupWebsocket(ctx context.Context, client *http.Client, addr *jid.JID) (urls []string, err error) {
	return lookupEndpoint(ctx, client, addr, "ws")
}

// LookupBOSH discovers BOSH endpoints that are valid for the given address
// using DNS TXT records and Web Host Metadata as described in XEP-0156:
// Discovering Alternative XMPP Connection Methods. If client is nil, only DNS
// is queried.
func LookupBOSH(ctx context.Context, client *http.Client, addr *jid.JID) (urls []string, err error) {
	return lookupEndpoint(ctx, client, addr, "bosh")
}

func lookupEndpoint(ctx context.Context, client *http.Client, addr *jid.JID, conntype string) (urls []string, err error) {
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
		default:
			panic("xmpp.lookupHostMeta: Invalid conntype specified")
		}
	}

	return urls, err
}

// TODO(ssw): Memoize the following functions?

func lookupHostMeta(ctx context.Context, client *http.Client, name, conntype string) (urls []string, err error) {
	url, err := url.Parse(name)
	if err != nil {
		return urls, err
	}
	url.Path = ""

	ctx, cancel := context.WithCancel(ctx)

	var xrd *internal.XRD
	// Race! If one of the two goroutines does not error, we want that one. If
	// both error, or both are error free, we don't care.
	go func() {
		defer cancel()
		x, e := getHostMetaXML(ctx, client, url.String())
		if e != nil {
			err = e
			return
		}
		xrd, err = &x, e
	}()
	go func() {
		defer cancel()
		x, e := getHostMetaJSON(ctx, client, url.String())
		if e != nil {
			err = e
			return
		}
		xrd, err = &x, e
	}()

	if xrd == nil {
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
		default:
			panic("xmpp.lookupHostMeta: Invalid conntype specified")
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

func getHostMetaJSON(
	ctx context.Context, client *http.Client, name string) (xrd internal.XRD, err error) {
	resp, err := ctxhttp.Get(ctx, client, path.Join(name, hostMetaJSON))
	if err != nil {
		return xrd, err
	}

	if _, ok := <-ctx.Done(); ok {
		return xrd, ctx.Err()
	}

	d := json.NewDecoder(resp.Body)

	// TODO: We should probably tokenize this and have the ability to cancel
	// anywhere in between.
	err = d.Decode(&xrd)
	return xrd, err
}
