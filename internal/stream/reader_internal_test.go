// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stream

import (
	"encoding/xml"
	"math"
	"testing"

	"mellium.im/xmlstream"
)

func TestMaxDepthErrorReader(t *testing.T) {
	r := errorReader{r: xmlstream.ReaderFunc(func() (xml.Token, error) {
		return xml.StartElement{Name: xml.Name{Local: "foo"}}, nil
	})}

	r.depth = math.MaxUint64
	_, err := r.Token()
	if err != errMaxNesting {
		t.Errorf("unexpected error: want=%v, got=%v", errMaxNesting, err)
	}
}
