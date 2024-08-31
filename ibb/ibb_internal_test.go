// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibb

import (
	"context"

	"mellium.im/xmpp/jid"
)

// WaitExpect blocks until Expect has been called for the given JID/SID
// combination.
//
// This method is only available in tests and is a work around for issue #407.
// It's not ideal that we have to mess with the internals of the listener, but
// it was the only way I could think of to make sure that expect was actually
// listening before sending the IQ.
//
// DANGER:
// If you are modifying this test, make sure that this function does not modify
// any internal state of the listener.
// We want to be testing the actual code as it will be run and not have tests
// messing with things.
func (l *Listener) WaitExpect(ctx context.Context, from jid.JID, sid string) error {
	key := from.String() + ":" + sid
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			l.eLock.Lock()
			_, ok := l.expected[key]
			l.eLock.Unlock()
			if ok {
				return nil
			}
		}
	}
}
