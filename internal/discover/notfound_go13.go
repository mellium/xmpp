// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// +build go1.13

package discover

import (
	"net"
)

func isNotFound(dnsErr *net.DNSError) bool {
	return dnsErr.IsNotFound
}
