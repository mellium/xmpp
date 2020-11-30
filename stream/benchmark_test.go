// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream_test

import (
	"net"
	"testing"

	"mellium.im/xmpp/stream"
)

func BenchmarkSeeOtherHostError(b *testing.B) {
	ip := &net.IPAddr{IP: net.ParseIP("2001:db8::68")}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = stream.SeeOtherHostError(ip)
	}
}
