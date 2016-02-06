// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package errors

import (
	"encoding/xml"
	"fmt"
)

func ExampleNew() {
	e := New(
		xml.Name{"http://example.net", "comedy"},
		"There was a comedy of errors.",
	)
	b, _ := xml.Marshal(e)
	fmt.Println(string(b))
	// Output:
	// <comedy xmlns="http://example.net">There was a comedy of errors.</comedy>
}
