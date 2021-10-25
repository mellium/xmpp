// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genfeature -vars "Feature:NS,FeatureNotify:NSNotify"

// Package bookmarks implements storing bookmarks to chat rooms.
package bookmarks // import "mellium.im/xmpp/bookmarks"

// Namespaces used by this package.
const (
	NS       = "urn:xmpp:bookmarks:1"
	NSNotify = "urn:xmpp:bookmarks:1+notify"
	NSCompat = "urn:xmpp:bookmarks:1#compat"
)
