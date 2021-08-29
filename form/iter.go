// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package form

// Iter is the interface implemented by types that implement disco form
// extensions.
type Iter interface {
	ForForms(node string, f func(*Data) error) error
}
