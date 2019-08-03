// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// +build !go1.13

package discover

import (
	"net"
	"strings"
)

func isNotFound(err error) bool {
	dnsErr, ok := err.(*net.DNSError)
	return ok && strings.Contains(dnsErr.Error(), "no such host")
}
