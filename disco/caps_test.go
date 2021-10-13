// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package disco

import (
	"crypto/sha1"
	"hash"
	"strconv"
	"testing"

	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/muc"
)

var verificationTestCases = [...]struct {
	info Info
	h    hash.Hash
	out  string
}{
	0: {
		info: Info{
			Identity: []info.Identity{{
				Type:     "pc",
				Category: "client",
				Name:     "Exodus 0.9.1",
			}},
			Features: []info.Feature{
				{Var: NSInfo},
				{Var: NSCaps},
				{Var: NSItems},
				{Var: muc.NS},
			},
		},
		h:   sha1.New(),
		out: `QgayPKawpkPSDYmwT/WM94uAlu0=`,
	},
	1: {
		info: Info{
			Identity: []info.Identity{
				{
					Lang:     "en",
					Type:     "pc",
					Category: "client",
					Name:     "Psi 0.11",
				},
				{
					Lang:     "el",
					Type:     "pc",
					Category: "client",
					Name:     "Ψ 0.11",
				},
			},
			Features: []info.Feature{
				{Var: NSInfo},
				{Var: NSCaps},
				{Var: NSItems},
				{Var: muc.NS},
			},
			Form: []form.Data{*form.New(
				form.Hidden("FORM_TYPE", form.Value("urn:xmpp:dataforms:softwareinfo")),
				form.TextMulti("ip_version", form.Value("ipv4"), form.Value("ipv6")),
				form.Text("os", form.Value("Mac")),
				form.Text("os_version", form.Value("10.5.1")),
				form.Text("software", form.Value("Psi")),
				form.Text("software_version", form.Value("0.11")),
			),
			}},
		h:   sha1.New(),
		out: `q07IKJEyjvHSyhy//CH0CxmKi8w=`,
	},
}

func TestVerification(t *testing.T) {
	for i, tc := range verificationTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := tc.info.AppendHash(nil, tc.h)

			if s := string(out); s != tc.out {
				t.Fatalf("wrong hash output: want=%s, got=%s", tc.out, s)
			}
		})
	}

}
