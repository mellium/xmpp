// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package discover

import (
	"net"
)

func isNotFound(err error) bool {
	dnsErr, ok := err.(*net.DNSError)
	return ok && dnsErr.IsNotFound
}
