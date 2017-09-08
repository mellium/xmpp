// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.
//
// Some code code in this file was copied from the Go crypto/x509 package:
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.GO file.

// Package x509 parses X.509-encoded keys and certificates.
package x509 // import "mellium.im/xmpp/x509"

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
)

var (
	oidExtensionSubjectAltName = []int{2, 5, 29, 17}
)

// Certificate represents an X.509 certificate with additional fields for XMPP
// use.
type Certificate struct {
	x509.Certificate

	SRVNames      []string
	XMPPAddresses []string
}

// FromCertificate parses the Subject Alternative Name from the provided
// x509.Certificate and creates a new Certificate with the extra fields
// populated.
func FromCertificate(crt *x509.Certificate) (*Certificate, error) {
	srvNames, xmppAddrs, err := parseSANExtensions(crt.Extensions)
	return &Certificate{
		Certificate:   *crt,
		SRVNames:      srvNames,
		XMPPAddresses: xmppAddrs,
	}, err
}

// ParseCertificate parses a single certificate from the given ASN.1 DER data.
func ParseCertificate(asn1Data []byte) (*Certificate, error) {
	crt, err := x509.ParseCertificate(asn1Data)
	if err != nil {
		return nil, err
	}
	srvNames, xmppAddrs, err := parseSANExtensions(crt.Extensions)
	return &Certificate{
		Certificate:   *crt,
		SRVNames:      srvNames,
		XMPPAddresses: xmppAddrs,
	}, err
}

func parseSANExtensions(extensions []pkix.Extension) (srvNames, xmppAddrs []string, err error) {
	for _, ext := range extensions {
		if !ext.Id.Equal(oidExtensionSubjectAltName) {
			// Not a SAN block
			continue
		}

		newNames, newXMPPAddrs, err := parseSANExtension(ext.Value)
		if err != nil {
			return srvNames, xmppAddrs, err
		}
		srvNames = append(srvNames, newNames...)
		xmppAddrs = append(xmppAddrs, newXMPPAddrs...)
	}

	return srvNames, xmppAddrs, err
}

func parseSANExtension(value []byte) (srvNames, xmppAddresses []string, err error) {
	// RFC 5280, 4.2.1.6

	// SubjectAltName ::= GeneralNames
	//
	// GeneralNames ::= SEQUENCE SIZE (1..MAX) OF GeneralName
	//
	// GeneralName ::= CHOICE {
	//      otherName                       [0]     OtherName,
	//      rfc822Name                      [1]     IA5String,
	//      dNSName                         [2]     IA5String,
	//      x400Address                     [3]     ORAddress,
	//      directoryName                   [4]     Name,
	//      ediPartyName                    [5]     EDIPartyName,
	//      uniformResourceIdentifier       [6]     IA5String,
	//      iPAddress                       [7]     OCTET STRING,
	//      registeredID                    [8]     OBJECT IDENTIFIER }
	var seq asn1.RawValue
	var rest []byte
	if rest, err = asn1.Unmarshal(value, &seq); err != nil {
		return
	} else if len(rest) != 0 {
		err = errors.New("xmpp/x509: trailing data after X.509 extension")
		return
	}
	if !seq.IsCompound || seq.Tag != 16 || seq.Class != 0 {
		err = asn1.StructuralError{Msg: "bad SAN sequence"}
		return
	}

	return parseRest(seq.Bytes)
}

func parseRest(rest []byte) (srvNames, xmppAddresses []string, err error) {
	for len(rest) > 0 {
		var v asn1.RawValue
		rest, err = asn1.Unmarshal(rest, &v)
		if err != nil {
			return
		}
		switch v.Tag {
		case 0:
			srvNew, xmppNew, err := parseRest(v.Bytes)
			if err != nil {
				return srvNames, xmppAddresses, err
			}
			srvNames = append(srvNames, srvNew...)
			xmppAddresses = append(xmppAddresses, xmppNew...)
		// TODO: We should probably verify that the OID matches.
		// case 6:
		case 12:
			xmppAddresses = append(xmppAddresses, string(v.Bytes))
		case 22:
			srvNames = append(srvNames, string(v.Bytes))
		}
	}
	return
}
