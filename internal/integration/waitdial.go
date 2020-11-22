// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package integration

import (
	"fmt"
	"net"
	"time"
)

func waitSocket(network, socket string) error {
	timeout := time.Second
	for connAttempts := 10; connAttempts > 0; connAttempts-- {
		time.Sleep(timeout)
		timeout += 500 * time.Millisecond
		conn, err := net.DialTimeout(network, socket, timeout)
		if err != nil {
			continue
		}
		if err = conn.Close(); err != nil {
			return fmt.Errorf("failed to close probe connection: %w", err)
		}
		return nil
	}
	return fmt.Errorf("failed to bind to %s", socket)
}
