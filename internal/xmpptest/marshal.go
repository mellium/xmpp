// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpptest

import (
	"encoding/xml"
	"errors"
	"reflect"
	"strconv"
	"testing"
)

// EncodingTestCase is a test that marshals the value and checks that the result
// matches XML, then unmarshals XML into a new zero value of the type in value
// and checks that it matches the original value with reflect.DeepEqual.
// If NoMarshal or NoUnmarshal is set then the corresponding part of the test is
// not run (for payloads that are not roundtrippable).
type EncodingTestCase struct {
	Value       interface{}
	XML         string
	Err         error
	NoMarshal   bool
	NoUnmarshal bool
}

// RunEncodingTests iterates over the test cases and runs each one.
func RunEncodingTests(t *testing.T, testCases []EncodingTestCase) {
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if !tc.NoMarshal {
				t.Run("marshal", func(t *testing.T) {
					x, err := xml.Marshal(tc.Value)
					if !errors.Is(err, tc.Err) {
						t.Fatalf("unexpected error: want=%v, got=%v", tc.Err, err)
					}
					if out := string(x); out != tc.XML {
						t.Fatalf("unexpected output:\nwant=%q,\n got=%q", tc.XML, out)
					}
				})
			}
			if !tc.NoUnmarshal {
				t.Run("unmarshal", func(t *testing.T) {
					valType := reflect.TypeOf(tc.Value).Elem()
					newVal := reflect.New(valType).Interface()
					err := xml.Unmarshal([]byte(tc.XML), &newVal)
					if !errors.Is(err, tc.Err) {
						t.Fatalf("unexpected error: want=%v, got=%v", tc.Err, err)
					}
					if !reflect.DeepEqual(newVal, tc.Value) {
						t.Fatalf("unexpected value: want=%+v, got=%+v", tc.Value, newVal)
					}
				})
			}
		})
	}
}
