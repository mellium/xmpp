// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package muc_test

import (
	"encoding/xml"

	"mellium.im/xmpp/muc"
)

var (
	_ xml.MarshalerAttr   = (*muc.Role)(nil)
	_ xml.UnmarshalerAttr = (*muc.Role)(nil)
	_ xml.MarshalerAttr   = (*muc.Affiliation)(nil)
	_ xml.UnmarshalerAttr = (*muc.Affiliation)(nil)
)
