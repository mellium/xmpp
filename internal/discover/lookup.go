// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package discover is used to look up information about XMPP-based services.
package discover

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
	"mellium.im/xmpp/jid"
)

const (
	wsPrefix     = "_xmpp-client-websocket="
	boshPrefix   = "_xmpp-client-xbosh="
	wsRel        = "urn:xmpp:alt-connections:websocket"
	boshRel      = "urn:xmpp:alt-connections:xbosh"
	hostMetaXML  = "/.well-known/host-meta"
	wsConnType   = "ws"
	boshConnType = "bosh"
)

// XRD represents an Extensible Resource Descriptor document of the form:
//
//    <?xml version='1.0' encoding=utf-9'?>
//    <XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>
//      …
//      <Link rel="urn:xmpp:alt-connections:xbosh"
//            href="https://web.example.com:5280/bosh" />
//      <Link rel="urn:xmpp:alt-connections:websocket"
//            href="wss://web.example.com:443/ws" />
//      …
//    </XRD>
//
// as defined by RFC 6415 and OASIS.XRD-1.0.
type XRD struct {
	XMLName xml.Name `xml:"http://docs.oasis-open.org/ns/xri/xrd-1.0 XRD"`
	Links   []Link   `xml:"Link"`
}

// Link is an individual hyperlink in an XRD document.
type Link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

// Errors related to address and service lookups.
var (
	ErrNoServiceAtAddress = errors.New("This address does not offer the requested service")
)

var (
	xrdName = xml.Name{
		Space: "http://docs.oasis-open.org/ns/xri/xrd-1.0",
		Local: "XRD",
	}
)

// LookupPort returns the default port for the provided network and service
// using net.LookupPort.
// If the provided service is one of xmpp-client, xmpp-server, or xmpp-bosh and
// it is not found by net.LookupPort, a default value is returned.
func LookupPort(network, service string) (uint16, error) {
	p, err := net.LookupPort(network, service)
	if err == nil {
		return uint16(p), err
	}
	switch service {
	case "xmpp-client", "xmpps-client":
		return 5222, nil
	case "xmpp-server", "xmpps-server":
		return 5269, nil
	case "xmpp-bosh":
		return 5280, nil
	}
	return 0, err
}

// LookupService looks for an XMPP service hosted by the given address. It
// returns addresses from SRV records or the default domain (as a fake SRV
// record) if no real records exist. Service should be one of "xmpp-client" or
// "xmpp-server".
func LookupService(ctx context.Context, resolver *net.Resolver, service, network string, addr net.Addr) (addrs []*net.SRV, err error) {
	switch j := addr.(type) {
	case nil:
		return nil, nil
	case *jid.JID:
		addr = j.Domain()
	case jid.JID:
		addr = j.Domain()
	}
	_, addrs, err = resolver.LookupSRV(ctx, service, "tcp", addr.String())
	if dnsErr, ok := err.(*net.DNSError); (ok && !isNotFound(dnsErr)) || (!ok && err != nil) {
		return addrs, err
	}
	err = nil

	// RFC 6230 §3.2.1
	//    3.  If a response is received, it will contain one or more
	//        combinations of a port and FDQN, each of which is weighted and
	//        prioritized as described in [DNS-SRV].  (However, if the result
	//        of the SRV lookup is a single resource record with a Target of
	//        ".", i.e., the root domain, then the initiating entity MUST abort
	//        SRV processing at this point because according to [DNS-SRV] such
	//        a Target "means that the service is decidedly not available at
	//        this domain".)
	if len(addrs) == 1 && addrs[0].Target == "." {
		return nil, ErrNoServiceAtAddress
	}

	return addrs, nil
}

// LookupWebsocket discovers websocket endpoints that are valid for the given
// address using DNS TXT records and Web Host Metadata as described in XEP-0156.
// If client is nil, only DNS is queried.
func LookupWebsocket(ctx context.Context, resolver *net.Resolver, client *http.Client, addr *jid.JID) (urls []string, err error) {
	return lookupEndpoint(ctx, resolver, client, addr, wsConnType)
}

// LookupBOSH discovers BOSH endpoints that are valid for the given address
// using DNS TXT records and Web Host Metadata as described in XEP-0156. If
// client is nil, only DNS is queried.
func LookupBOSH(ctx context.Context, resolver *net.Resolver, client *http.Client, addr *jid.JID) (urls []string, err error) {
	return lookupEndpoint(ctx, resolver, client, addr, boshConnType)
}

func lookupEndpoint(ctx context.Context, resolver *net.Resolver, client *http.Client, addr *jid.JID, conntype string) (urls []string, err error) {
	if conntype != wsConnType && conntype != boshConnType {
		panic("xmpp.lookupEndpoint: Invalid conntype specified")
	}

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
		urls, err = lookupDNS(ctx, resolver, name, conntype)
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

func lookupDNS(ctx context.Context, resolver *net.Resolver, name, conntype string) (urls []string, err error) {
	if conntype != wsConnType && conntype != boshConnType {
		panic("xmpp.lookupEndpoint: Invalid conntype specified")
	}

	txts, err := resolver.LookupTXT(ctx, name)
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
		case wsConnType:
			if s = strings.TrimPrefix(txt, wsPrefix); s != txt {
				urls = append(urls, s)
			}
		case boshConnType:
			if s = strings.TrimPrefix(txt, boshPrefix); s != txt {
				urls = append(urls, s)
			}
		}
	}

	return urls, err
}

// TODO(ssw): Memoize the following functions?

func lookupHostMeta(ctx context.Context, client *http.Client, name, conntype string) (urls []string, err error) {
	if conntype != wsConnType && conntype != boshConnType {
		panic("xmpp.lookupEndpoint: Invalid conntype specified")
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
		case wsConnType:
			if link.Rel == wsRel {
				urls = append(urls, link.Href)
			}
		case boshConnType:
			if link.Rel == boshRel {
				urls = append(urls, link.Href)
			}
		}
	}
	return urls, err
}

func getHostMetaXML(
	ctx context.Context, client *http.Client, name string) (xrd XRD, err error) {
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
