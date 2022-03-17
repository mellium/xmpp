// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package compress implements XEP-0138: Stream Compression and XEP-0229: Stream
// Compression with LZW.
//
// Be advised: stream compression has many of the same security considerations
// as TLS compression (see RFC3749 §6) and may be difficult to implement safely
// without special expertise.
//
// Deprecated: the stream compression implemented in this package is no longer
// recommended by the XSF and will be removed in a future version of this
// library.
package compress // import "mellium.im/xmpp/compress"

import (
	"mellium.im/legacy/compress"
	"mellium.im/xmpp"
)

// Namespaces used by stream compression.
const (
	NSFeatures = compress.NSFeatures
	NSProtocol = compress.NSProtocol
)

// New returns a new xmpp.StreamFeature that can be used to negotiate stream
// compression.
// The returned stream feature always supports ZLIB compression; other
// compression methods are optional.
func New(methods ...Method) xmpp.StreamFeature {
	return compress.New(methods...)
}

var (
	// LZW implements stream compression using the Lempel-Ziv-Welch (DCLZ)
	// compressed data format.
	LZW Method = compress.LZW
)

// Method is a stream compression method.
// Custom methods may be defined, but generally speaking the only supported
// methods will be those with names defined in the "Stream Compression Methods
// Registry" maintained by the XSF Editor:
// https://xmpp.org/registrar/compress.html
//
// Since ZLIB is always supported, a Method is not defined for it.
type Method = compress.Method
