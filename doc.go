// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package jid implements XMPP addresses (JIDs) as described in RFC 6122.
// The syntax for a JID is defined as follows using the Augmented Backus-Naur
// Form:
//
//      jid           = [ localpart "@" ] domainpart [ "/" resourcepart ]
//      localpart     = 1*(nodepoint)
//                      ;
//                      ; a "nodepoint" is a UTF-8 encoded Unicode code
//                      ; point that satisfies the Nodeprep profile of
//                      ; stringprep
//                      ;
//      domainpart    = IP-literal / IPv4address / ifqdn
//                      ;
//                      ; the "IPv4address" and "IP-literal" rules are
//                      ; defined in RFC 3986, and the first-match-wins
//                      ; (a.k.a. "greedy") algorithm described in RFC
//                      ; 3986 applies to the matching process
//                      ;
//                      ; note well that reuse of the IP-literal rule
//                      ; from RFC 3986 implies that IPv6 addresses are
//                      ; enclosed in square brackets (i.e., beginning
//                      ; with '[' and ending with ']'), which was not
//                      ; the case in RFC 3920
//                      ;
//      ifqdn         = 1*(namepoint)
//                      ;
//                      ; a "namepoint" is a UTF-8 encoded Unicode
//                      ; code point that satisfies the Nameprep
//                      ; profile of stringprep
//                      ;
//      resourcepart  = 1*(resourcepoint)
//                      ;
//                      ; a "resourcepoint" is a UTF-8 encoded Unicode
//                      ; code point that satisfies the Resourceprep
//                      ; profile of stringprep
//                      ;
package jid
