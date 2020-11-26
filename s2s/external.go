// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package s2s

import (
	"mellium.im/sasl"
)

// TLSAuth returns a SASL mechanism that requests that the remove server
// authenticate the connection using the TLS client certificate.
// This is an implementation of SASL EXTERNAL specifically tailored to XMPP.
func TLSAuth() sasl.Mechanism {
	return sasl.Mechanism{
		Name: "EXTERNAL",
		Start: func(m *sasl.Negotiator) (bool, []byte, interface{}, error) {
			_, _, identity := m.Credentials()
			return false, identity, nil, nil
		},
		Next: func(m *sasl.Negotiator, challenge []byte, _ interface{}) (bool, []byte, interface{}, error) {
			// If we're a client, or we're a server that's past the AuthTextSent step,
			// we should never actually hit this step.
			if m.State()&sasl.Receiving == 0 || m.State()&sasl.StepMask != sasl.AuthTextSent {
				return false, nil, nil, sasl.ErrTooManySteps
			}

			panic("tls auth not yet implemented for receiving connections")
		},
	}
}
