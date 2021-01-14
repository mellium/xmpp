// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

// Errors used in the SASL package that are needed in tests (but should not be
// exported outside of testing).
var (
	ErrNoMechanisms      = errNoMechanisms
	ErrUnexpectedPayload = errUnexpectedPayload
	ErrTerminated        = errTerminated
)
