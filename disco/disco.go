// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run gen.go
//go:generate go run ../internal/genfeature -filename features.go -receiver "h *discoHandler" -vars Feature:NSInfo

// Package disco implements service discovery.
package disco // import "mellium.im/xmpp/disco"

// Namespaces used by this package.
const (
	NSInfo  = `http://jabber.org/protocol/disco#info`
	NSItems = `http://jabber.org/protocol/disco#items`
	NSCaps  = `http://jabber.org/protocol/caps`
)
