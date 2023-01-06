// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run ../internal/genpubsub
//go:generate go run -tags=tools golang.org/x/tools/cmd/stringer -output=string.go -type=SubType,Condition,Feature -linecomment

// Package pubsub implements data storage using a publish–subscribe pattern.
package pubsub // import "mellium.im/xmpp/pubsub"

// Various namespaces used by this package, provided as a convenience.
const (
	NS        = `http://jabber.org/protocol/pubsub`
	NSErrors  = `http://jabber.org/protocol/pubsub#errors`
	NSEvent   = `http://jabber.org/protocol/pubsub#event`
	NSOptions = `http://jabber.org/protocol/pubsub#subscription-options`
	NSOwner   = `http://jabber.org/protocol/pubsub#owner`
	NSPaging  = `http://jabber.org/protocol/pubsub#rsm`
)
