// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package attr

import (
	"encoding/xml"
)

// Get returns the value and index of the first attribute with the provided
// local name from a list of attributes or -1 and an empty string if no such
// attribute exists.
func Get(attr []xml.Attr, local string) (int, string) {
	for idx, a := range attr {
		if a.Name.Local == local {
			return idx, a.Value
		}
	}
	return -1, ""
}
