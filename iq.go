// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
)

// IQ ("Information Query") is used as a general request response mechanism.
// IQ's are one-to-one, provide get and set semantics, and always require a
// response in the form of a result or an error.
type IQ struct {
	stanza

	XMLName xml.Name `xml:"iq"`
}

type iqType int

const (
	// A GetIQ is used to query another entity for information.
	GetIQ iqType = iota

	// A SetIQ is used to provide data to another entity, set new values, replace
	// existing values, and other such operations.
	SetIQ

	// A ResultIQ is sent in response to a successful GetIQ or SetIQ stanza.
	ResultIQ

	// An ErrorIQ is sent to report that an error occured during the delivery or
	// processing of a GetIQ or SetIQ.
	ErrorIQ
)
