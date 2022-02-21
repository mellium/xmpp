// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package crypto

import (
	"fmt"

	"mellium.im/xmpp/disco/info"
)

// Features returns an iter that can be registered against a mux to advertise
// support for the hash list.
// The iter will return an error for any hashes that are not available in the
// binary.
func Features(h ...Hash) info.FeatureIter {
	return handler(h)
}

type handler []Hash

func (h handler) ForFeatures(node string, f func(info.Feature) error) error {
	if node != "" {
		return nil
	}
	for _, h := range h {
		if !h.Available() {
			return fmt.Errorf("%w %s", ErrUnlinkedAlgo, h.String())
		}
		ns, err := h.Namespace()
		if err != nil {
			return err
		}
		err = f(info.Feature{Var: ns})
		if err != nil {
			return err
		}
	}
	return f(Feature)
}
