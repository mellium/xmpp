// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stanza

import (
	"fmt"
)

var (
	_ fmt.Stringer = (*presenceType)(nil)
	_ fmt.Stringer = ProbePresence
)
