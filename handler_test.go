// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp_test

import (
	"encoding/xml"
	"errors"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
)

var errHandlerFuncSentinal = errors.New("handler test")

type sentinalReadWriter struct{}

func (sentinalReadWriter) Token() (xml.Token, error)                         { return nil, nil }
func (sentinalReadWriter) EncodeToken(xml.Token) error                       { return nil }
func (sentinalReadWriter) Encode(interface{}) error                          { return nil }
func (sentinalReadWriter) EncodeElement(interface{}, xml.StartElement) error { return nil }

func TestHandlerFunc(t *testing.T) {
	s := &xml.StartElement{}
	var f xmpp.HandlerFunc = func(r xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		if _, ok := r.(sentinalReadWriter); !ok {
			t.Errorf("HandleXMPP did not pass reader to HandlerFunc")
		}
		if start != s {
			t.Errorf("HandleXMPP did not pass start token to HandlerFunc")
		}
		return errHandlerFuncSentinal
	}

	err := f.HandleXMPP(sentinalReadWriter{}, s)
	if err != errHandlerFuncSentinal {
		t.Errorf("HandleXMPP did not return handlerfunc error, got %q", err)
	}
}
