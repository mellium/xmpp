// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package jid

// Unsafe is a JID that has not had any normalization, length checks, UTF-8
// validation, or other safety measures applied.
//
// It can be a source of bugs, or even a security risk, if used improperly.
type Unsafe struct {
	JID
}

// NewUnsafe constructs a new unsafe JID.
// For more information, see the Unsafe type.
func NewUnsafe(localpart, domainpart, resourcepart string) Unsafe {
	data := make([]byte, 0, len(localpart)+len(domainpart)+len(resourcepart))
	data = append(data, []byte(localpart)...)
	data = append(data, []byte(domainpart)...)
	data = append(data, []byte(resourcepart)...)
	return Unsafe{
		JID: JID{
			locallen:  len(localpart),
			domainlen: len(domainpart),
			data:      data,
		},
	}
}

// ParseUnsafe constructs a new unsafe JID from a string.
// For more information, see the Unsafe type.
func ParseUnsafe(s string) (Unsafe, error) {
	localpart, domainpart, resourcepart, err := splitString(s, false)
	return NewUnsafe(localpart, domainpart, resourcepart), err
}
