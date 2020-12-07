// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package form_test

import (
	"mellium.im/xmpp/form"
)

var (
	_ form.Field = form.Boolean{}
	_ form.Field = form.Fixed{}
	_ form.Field = form.Hidden{}
	_ form.Field = form.JIDMulti{}
	_ form.Field = form.JID{}
	_ form.Field = form.ListMulti{}
	_ form.Field = form.List{}
	_ form.Field = form.TextMulti{}
	_ form.Field = form.Text{}
)
