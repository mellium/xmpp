// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package pubsub implements data storage using a publish–subscribe pattern.
//
// This package is currently in internal while the API is flushed out as part of
// work on other higher-level packages that use it under the hood.
// It will eventually be moved to the exported packages once enough
// functionality is implemented and the API is somewhat more stable.
package pubsub // import "mellium.im/xmpp/internal/pubsub"

// Various namespaces used by this package, provided as a convenience.
const (
	NS       = `http://jabber.org/protocol/pubsub`
	NSPaging = `http://jabber.org/protocol/pubsub#rsm`
)
