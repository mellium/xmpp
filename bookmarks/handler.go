// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package bookmarks

import (
	"mellium.im/xmpp/disco/info"
)

// Handler can be registered against a mux to handle bookmark pushes.
type Handler struct {
}

// ForFeatures implements info.FeatureIter.
func (h Handler) ForFeatures(node string, f func(info.Feature) error) error {
	if node != "" {
		return nil
	}

	err := f(FeatureNotify)
	if err != nil {
		return err
	}
	return f(Feature)
}
