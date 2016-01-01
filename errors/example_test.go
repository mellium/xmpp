// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package errors

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

func ExampleStreamError_UnmarshalXML() {
	b := bytes.NewBufferString(`<stream:error>
	<restricted-xml xmlns="urn:ietf:params:xml:ns:xmpp-streams"/>
</stream:error>`)

	d := xml.NewDecoder(b)
	s := &StreamError{}
	d.Decode(s)

	fmt.Println(s.Error())
	// Output: restricted-xml
}

func ExampleStreamError_MarshalXML() {

	b, _ := xml.MarshalIndent(NotAuthorized, "", "  ")
	fmt.Println(string(b))
	// Output:
	// <stream:error>
	//   <not-authorized xmlns="urn:ietf:params:xml:ns:xmpp-streams"></not-authorized>
	// </stream:error>
}
