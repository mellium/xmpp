package xmpp_test

import (
	"encoding/xml"
	"errors"
	"testing"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
)

var errHandlerFuncSentinal = errors.New("handler test")

type sentinalDecodeEncoder struct{}

func (sentinalDecodeEncoder) Token() (xml.Token, error)                          { return nil, nil }
func (sentinalDecodeEncoder) Decode(interface{}) error                           { return nil }
func (sentinalDecodeEncoder) DecodeElement(interface{}, *xml.StartElement) error { return nil }
func (sentinalDecodeEncoder) EncodeToken(xml.Token) error                        { return nil }
func (sentinalDecodeEncoder) Encode(interface{}) error                           { return nil }
func (sentinalDecodeEncoder) EncodeElement(interface{}, xml.StartElement) error  { return nil }

func TestHandlerFunc(t *testing.T) {
	s := &xml.StartElement{}
	var f xmpp.HandlerFunc = func(r xmlstream.DecodeEncoder, start *xml.StartElement) error {
		if _, ok := r.(sentinalDecodeEncoder); !ok {
			t.Errorf("HandleXMPP did not pass reader to HandlerFunc")
		}
		if start != s {
			t.Errorf("HandleXMPP did not pass start token to HandlerFunc")
		}
		return errHandlerFuncSentinal
	}

	err := f.HandleXMPP(sentinalDecodeEncoder{}, s)
	if err != errHandlerFuncSentinal {
		t.Errorf("HandleXMPP did not return handlerfunc error, got %q", err)
	}
}
