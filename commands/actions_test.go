// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package commands_test

import (
	"testing"

	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/internal/xmpptest"
)

func TestActions(t *testing.T) {
	xmpptest.RunEncodingTests(t, []xmpptest.EncodingTestCase{
		{
			Value: func() *commands.Actions {
				action := commands.Prev | commands.Next | commands.Complete
				return &action
			}(),
			XML: `<actions><prev></prev><next></next><complete></complete></actions>`,
		},
		{
			Value: func() *commands.Actions {
				action := commands.Next | commands.Complete | (commands.Next << 3)
				return &action
			}(),
			XML: `<actions execute="next"><next></next><complete></complete></actions>`,
		},
		{
			Value: func() *commands.Actions {
				action := commands.Next | commands.Complete | (commands.Prev << 3)
				return &action
			}(),
			XML: `<actions execute="prev"><next></next><complete></complete></actions>`,
		},
	})
}
