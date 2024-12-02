// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package discover is used to look up information about XMPP-based services.
package discover // import "mellium.im/xmpp/internal/discover"

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"

	"mellium.im/xmpp/jid"
)

const (
	wsRel        = "urn:xmpp:alt-connections:websocket"
	boshRel      = "urn:xmpp:alt-connections:xbosh"
	hostMetaXML  = "/.well-known/host-meta"
	wsConnType   = "ws"
	boshConnType = "bosh"
)

// XRD represents an Extensible Resource Descriptor document of the form:
//
//	<?xml version='1.0' encoding=utf-9'?>
//	<XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>
//	  …
//	  <Link rel="urn:xmpp:alt-connections:xbosh"
//	        href="https://web.example.com:5280/bosh" />
//	  <Link rel="urn:xmpp:alt-connections:websocket"
//	        href="wss://web.example.com:443/ws" />
//	  …
//	</XRD>
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

func isNotFound(err error) bool {
	var dnsErr *net.DNSError
	ok := errors.As(err, &dnsErr)
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
		}}
	case "xmpps-client":
		return []*net.SRV{{
			Target: domain,
			Port:   5223,
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
	return LookupServiceByDomain(ctx, resolver, service, addr.Domainpart())
}

// LookupServiceByDomain behaves exactly the same as LookupService, besides
// that the domain it tries to connect to is given as argument instead of
// using the domainpart of the JID.
func LookupServiceByDomain(ctx context.Context, resolver *net.Resolver, service string, domain string) (addrs []*net.SRV, err error) {
	switch service {
	case "xmpp-client", "xmpp-server", "xmpps-client", "xmpps-server":
	default:
		return nil, ErrInvalidService
	}
	_, addrs, err = resolver.LookupSRV(ctx, service, "tcp", domain)
	if err != nil {
		if !isNotFound(err) {
			return nil, err
		}

		// Add a fallback to the JID.
		return FallbackRecords(service, domain), nil
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
// address using Web Host Metadata as described in RFC7395.
func LookupWebSocket(ctx context.Context, client *http.Client, addr jid.JID) (urls []string, err error) {
	return lookupHostMeta(ctx, client, addr.Domain().String(), wsConnType)
}

// LookupBOSH discovers BOSH endpoints that are valid for the given address
// using Web Host Metadata as described in XEP-0156.
func LookupBOSH(ctx context.Context, client *http.Client, addr jid.JID) (urls []string, err error) {
	return lookupHostMeta(ctx, client, addr.Domain().String(), boshConnType)
}

func lookupHostMeta(ctx context.Context, client *http.Client, name, conntype string) (urls []string, err error) {
	if conntype != wsConnType && conntype != boshConnType {
		panic("xmpp.lookupEndpoint: Invalid conntype specified")
	}

	url, err := url.Parse("https://" + path.Join(name, hostMetaXML))
	if err != nil {
		return urls, err
	}

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

func getHostMetaXML(ctx context.Context, client *http.Client, name string) (xrd XRD, err error) {
	req, err := http.NewRequest("GET", name, nil)
	if err != nil {
		return xrd, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return xrd, err
	}
	/* #nosec */
	defer resp.Body.Close()
	// If the server sends us a lot of data it's probably good to just error out.
	body := io.LimitReader(resp.Body, http.DefaultMaxHeaderBytes)
	err = xml.NewDecoder(body).Decode(&xrd)
	return xrd, err
}
