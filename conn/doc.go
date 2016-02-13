// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package conn is a generic connection handler for client-to-server (C2S) and
// server-to-server (S2S) connections between XMPP endpoints. The conn package
// merely facilitates creating the underlying connections over which XMPP will
// be routed, and does not handle any XML itself.
//
// Be advised: This API is still unstable and is subject to change.
package conn // import "bitbucket.org/mellium/xmpp/conn"
