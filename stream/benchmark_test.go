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
		_ = stream.SeeOtherHostError(ip, nil)
	}
}
