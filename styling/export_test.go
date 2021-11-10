// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build export
// +build export

package styling_test

// This file is a tool that exports test data in JSON format for other libraries
// and languages to use.
// Running "go test -tags export" will cause the following TestMain function to
// run which will spit out the tests to standard out.

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"

	"mellium.im/xmpp/styling"
)

type jsonStyle struct {
	Mask  styling.Style
	Data  string
	Info  string
	Quote uint
}

type jsonTestCase struct {
	Name   string
	Input  string
	Tokens []jsonStyle
}

func TestMain(m *testing.M) {
	var outName = "decoder_tests.json"
	flag.StringVar(&outName, "export", outName, "a filename to export JSON tests to")
	flag.Parse()

	fd, err := os.Create(outName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open file: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := fd.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close file: %v", err)
			os.Exit(1)
		}
	}()

	var jsonTestCases []jsonTestCase
	for _, tc := range decoderTestCases {
		newTC := jsonTestCase{
			Name:  tc.name,
			Input: tc.input,
		}
		for _, tok := range tc.toks {
			newTC.Tokens = append(newTC.Tokens, jsonStyle{
				Mask:  tok.Mask,
				Data:  string(tok.Data),
				Info:  string(tok.Info),
				Quote: tok.Quote,
			})
		}
		jsonTestCases = append(jsonTestCases, newTC)
	}

	e := json.NewEncoder(fd)
	e.SetIndent("", "\t")
	err = e.Encode(jsonTestCases)
	if err != nil {
		panic(err)
	}
}
