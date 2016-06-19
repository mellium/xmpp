// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"errors"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

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
	ErrNoServiceAtAddress = errors.New("This address does not offer the requested service")
)

var (
	xrdName = xml.Name{
		Space: "http://docs.oasis-open.org/ns/xri/xrd-1.0",
		Local: "XRD",
	}
)

func lookupPort(network, service string) (int, error) {
	p, err := net.LookupPort(network, service)
	if err == nil {
		return p, err
	}
	switch service {
	case "xmpp-client":
		return 5222, nil
	case "xmpp-server":
		return 5269, nil
	case "xmpp-bosh":
		return 5280, nil
	}
	return 0, err
}

// lookupService looks for an XMPP service hosted by the given address. It
// returns addresses from SRV records or the default domain (as a fake SRV
// record) if no real records exist. Service should be one of "xmpp-client" or
// "xmpp-server".
func lookupService(service, network string, addr net.Addr) (addrs []*net.SRV, err error) {
	switch j := addr.(type) {
	case *jid.JID:
		addr = j.Domain()
	}
	_, addrs, err = net.LookupSRV(service, "tcp", addr.String())

	// RFC 6230 §3.2.1
	//    3.  If a response is received, it will contain one or more
	//        combinations of a port and FDQN, each of which is weighted and
	//        prioritized as described in [DNS-SRV].  (However, if the result
	//        of the SRV lookup is a single resource record with a Target of
	//        ".", i.e., the root domain, then the initiating entity MUST abort
	//        SRV processing at this point because according to [DNS-SRV] such
	//        a Target "means that the service is decidedly not available at
	//        this domain".)
	if err == nil && len(addrs) == 1 && addrs[0].Target == "." {
		return addrs, err
	}

	// Use domain and default port.
	p, err := lookupPort(network, service)
	if err != nil {
		return nil, err
	}
	addrs = []*net.SRV{{
		Target: addr.String(),
		Port:   uint16(p),
	}}
	return addrs, nil
}

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

	var (
		u  []string
		e  error
		wg sync.WaitGroup

		name = addr.Domainpart()
	)

	ctx, cancel := context.WithCancel(ctx)

	wg.Add(1)
	go func() {
		defer func() {
			if err == nil && len(urls) > 0 {
				cancel()
			}
			wg.Done()
		}()
		urls, err = lookupDNS(ctx, name, conntype)
	}()
	if client != nil {
		wg.Add(1)
		go func() {
			defer func() {
				if e == nil && len(u) > 0 {
					cancel()
				}
				wg.Done()
			}()
			u, e = lookupHostMeta(ctx, client, name, conntype)
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
	select {
	case <-ctx.Done():
		return urls, ctx.Err()
	default:
	}

	txts, err := net.LookupTXT(name)
	if err != nil {
		return urls, err
	}

	var s string
	for _, txt := range txts {
		select {
		case <-ctx.Done():
			return urls, ctx.Err()
		default:
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
	select {
	case <-ctx.Done():
		return urls, ctx.Err()
	default:
	}

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
