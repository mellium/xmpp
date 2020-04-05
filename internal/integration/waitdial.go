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
	connAttempts := 10
	timeout := time.Second
	for {
		if connAttempts--; connAttempts == 0 {
			return fmt.Errorf("failed to bind to %s", socket)
		}
		time.Sleep(timeout)
		conn, err := net.DialTimeout(network, socket, timeout)
		if err != nil {
			continue
		}
		timeout += 500 * time.Millisecond
		if err = conn.Close(); err != nil {
			return fmt.Errorf("failed to close probe connection: %w", err)
		}
		return nil
	}
}
