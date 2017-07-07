// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package internal

import (
	"encoding/xml"
)

func GetAttr(attr []xml.Attr, local string) string {
	for _, a := range attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}
