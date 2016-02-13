// Copyright 2015 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package stream

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"bitbucket.org/mellium/xmpp"
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

func ExampleUndefinedConditionError() {
	apperr := xmpp.Error{xml.Name{"http://example.org/ns", "app-error"}, ""}
	e := UndefinedConditionError(apperr)
	b, _ := xml.MarshalIndent(e, "", "  ")
	fmt.Println(string(b))
	// Output:
	// <stream:error>
	//   <undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-streams"><app-error xmlns="http://example.org/ns"></app-error></undefined-condition>
	// </stream:error>
}

func ExampleUndefinedConditionError_errorf() {
	apperr := fmt.Errorf("Unknown error")
	e := UndefinedConditionError(apperr)
	b, _ := xml.MarshalIndent(e, "", "  ")
	fmt.Println(string(b))
	// Output:
	// <stream:error>
	//   <undefined-condition xmlns="urn:ietf:params:xml:ns:xmpp-streams">Unknown error</undefined-condition>
	// </stream:error>
}
