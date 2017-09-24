// Copyright 2014 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package xmpp provides functionality from the Extensible Messaging and
// Presence Protocol.
//
// It is subdivided into several packages; this package provides functionality
// for establishing an XMPP session, feature negotiation (including an API for
// defining your own stream features), and low-level connection and stream
// manipulation. It allows sending and receiving "raw" XML over the stream but
// there is no special support for stanzas, and RFC 6120 semantics are not
// enforced. The jid package provides an implementation of the XMPP address
// format defined in RFC 7622. Several other packages also exist which provide
// functionality for common XMPP extensions. In future, a new package may
// provide a higher level API around sending / receiving stanzas, enforcing RFC
// 6120 and RFC 6121 semantics, etc.
//
// Be advised: This API is still unstable and is subject to change.
package xmpp // import "mellium.im/xmpp"

//go:generate stringer -type=iqType,errorType,messageType,presenceType -output stanzatype_string.go
