// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package discover is used to look up information about XMPP-based services.
package discover // import "mellium.im/xmpp/internal/discover"

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
	Links   []Link   `xml:"Link" json:"links"`
}

// Link is an individual hyperlink in an XRD document.
type Link struct {
	Rel  string `xml:"rel,attr" json:"rel"`
	Href string `xml:"href,attr" json:"href"`
}

var (
	xrdName = xml.Name{
		Space: "http://docs.oasis-open.org/ns/xri/xrd-1.0",
		Local: "XRD",
	}
)

// LookupPort returns the default port for the provided network and service
// using net.LookupPort.
// If the provided service is one of xmpp[s]-client, xmpp[s]-server, or
// xmpp-bosh and it is not found by net.LookupPort, a default value is returned.
func LookupPort(network, service string) (uint16, error) {
	p, err := net.LookupPort(network, service)
	if err == nil {
		return uint16(p), err
	}
	switch service {
	case "xmpps-client":
		// This port isn't actually registered with IANA for XMPP use, but for
		// historical reasons it's widely used for implicit TLS.
		return 5223, nil
	case "xmpp-client":
		return 5222, nil
	case "xmpp-server":
		return 5269, nil
	case "xmpps-server":
		// This port isn't actually registered with IANA for XMPP use, but for
		// historical reasons it's widely used for implicit TLS.
		return 5270, nil
	case "xmpp-bosh":
		return 5280, nil
	}
	return 0, err
}

func isNotFound(err error) bool {
	dnsErr, ok := err.(*net.DNSError)
	return ok && dnsErr.IsNotFound
}

// Errors returned by this package.
var (
	ErrInvalidService = errors.New("service must be one of xmpp[s]-client or xmpp[s]-server")
)

// FallbackRecords returns fake SRV records based on the service that can be
// used if no actual SRV records can be found but we believe that an XMPP
// service exists at the given domain.
func FallbackRecords(service, domain string) []*net.SRV {
	switch service {
	case "xmpp-client":
		return []*net.SRV{{
			Target: domain,
			Port:   5222,
		}, {
			Target: domain,
			Port:   80,
		}}
	case "xmpps-client":
		return []*net.SRV{{
			Target: domain,
			Port:   5223,
		}, {
			Target: domain,
			Port:   443,
		}, {
			Target: domain,
			Port:   5222,
		}}
	case "xmpp-server":
		return []*net.SRV{{
			Target: domain,
			Port:   5269,
		}}
	case "xmpps-server":
		return []*net.SRV{{
			Target: domain,
			Port:   5270,
		}, {
			Target: domain,
			Port:   5269,
		}}
	}
	return nil
}

// LookupService looks for an XMPP service hosted by the given address.
// It returns addresses from SRV records and if none are found returns several
// fallback records using the default domain of the JID and common ports on
// which XMPP servers listen for implicit TLS connections.
// If the target of the first record is "." it is removed and an empty list is
// returned.
// Service should be one of "xmpp[s]-client" or "xmpp[s]-server".
func LookupService(ctx context.Context, resolver *net.Resolver, service string, addr jid.JID) (addrs []*net.SRV, err error) {
	switch service {
	case "xmpp-client", "xmpp-server", "xmpps-client", "xmpps-server":
	default:
		return nil, ErrInvalidService
	}
	_, addrs, err = resolver.LookupSRV(ctx, service, "tcp", addr.Domainpart())
	if err != nil {
		if !isNotFound(err) {
			return nil, err
		}

		// Add a fallback to the JID.
		return FallbackRecords(service, addr.Domainpart()), nil
	}

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
		return nil, nil
	}
	return addrs, nil
}

// LookupWebSocket discovers websocket endpoints that are valid for the given
// address using DNS TXT records and Web Host Metadata as described in XEP-0156.
// If client is nil, only DNS is queried.
func LookupWebSocket(ctx context.Context, resolver *net.Resolver, client *http.Client, addr jid.JID) (urls []string, err error) {
	return lookupEndpoint(ctx, resolver, client, addr, wsConnType)
}

// LookupBOSH discovers BOSH endpoints that are valid for the given address
// using DNS TXT records and Web Host Metadata as described in XEP-0156.
// If client is nil, only DNS is queried.
func LookupBOSH(ctx context.Context, resolver *net.Resolver, client *http.Client, addr jid.JID) (urls []string, err error) {
	return lookupEndpoint(ctx, resolver, client, addr, boshConnType)
}

func lookupEndpoint(ctx context.Context, resolver *net.Resolver, client *http.Client, addr jid.JID, conntype string) (urls []string, err error) {
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
	if err != nil {
		return xrd, err
	}
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
