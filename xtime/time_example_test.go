// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xtime_test

import (
	"encoding/xml"
	"fmt"
	"time"

	"mellium.im/xmpp/xtime"
)

func ExampleTime() {
	t, _ := time.Parse(time.RFC3339, "2020-02-19T06:46:00-05:00")
	xt := xtime.Time{Time: t}

	o, _ := xml.Marshal(xt)
	fmt.Printf("%s\n", o)
	// Output:
	// <time xmlns="urn:xmpp:time"><tzo>-05:00</tzo><utc>2020-02-19T11:46:00Z</utc></time>
}
