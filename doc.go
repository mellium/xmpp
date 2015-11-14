// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package jid implements XMPP addresses (historically called "Jabber ID's" or
// "JID's") as described in RFC 7622.  The syntax for a JID is defined as
// follows using the Augmented Backus-Naur Form (ABNF) as specified in RFC
// 5234:
//
//     jid          = [ localpart "@" ] domainpart [ "/" resourcepart ]
//     localpart    = 1*1023(userbyte)
//                    ;
//                    ; a "userbyte" is a byte used to represent a
//                    ; UTF-8 encoded Unicode code point that can be
//                    ; contained in a string that conforms to the
//                    ; UsernameCaseMapped profile of the PRECIS
//                    ; IdentifierClass defined in RFC 7613
//                    ;
//     domainpart   = IP-literal / IPv4address / ifqdn
//                    ;
//                    ; the "IPv4address" and "IP-literal" rules are
//                    ; defined in RFCs 3986 and 6874, respectively,
//                    ; and the first-match-wins (a.k.a. "greedy")
//                    ; algorithm described in Appendix B of RFC 3986
//                    ; applies to the matching process
//                    ;
//     ifqdn        = 1*1023(domainbyte)
//                    ;
//                    ; a "domainbyte" is a byte used to represent a
//                    ; UTF-8 encoded Unicode code point that can be
//                    ; contained in a string that conforms to RFC 5890
//                    ;
//     resourcepart = 1*1023(opaquebyte)
//                    ;
//                    ; an "opaquebyte" is a byte used to represent a
//                    ; UTF-8 encoded Unicode code point that can be
//                    ; contained in a string that conforms to the
//                    ; OpaqueString profile of the PRECIS
//                    ; FreeformClass defined in RFC 7613
//                    ;
package jid
