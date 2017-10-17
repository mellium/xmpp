// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibr2

import (
	"mellium.im/xmpp/form"
)

// Form is a challenge that presents or receives a data form as specified in
// XEP-0004.
// If Form is used by a client, f is called and passed the form sent by the
// server.
// The returned form should be a response to the sent form.
// If Form is used by a server, f is called once with a nil form and should
// return a form to be sent to the client; it is then called again with the
// clients response at which point a nil form can be returned to terminate the
// exchange, or a second form to be sent to the client can be returned.
func Form(f func(data *form.Data) (*form.Data, error)) Challenge {
	return Challenge{
		Type: form.NS,
	}
}
