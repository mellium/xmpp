// Copyright 2017 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package x509_test

import (
	cryptox509 "crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"mellium.im/xmpp/x509"
)

type crtTest struct {
	crtData   string
	srvNames  []string
	xmppAddrs []string
	err       error
}

var crtTests = [...]crtTest{
	0: {
		xmppAddrs: []string{"example.org", "conference.example.org"},
		srvNames:  []string{"_xmpp-client.example.org", "_xmpp-server.example.org", "_xmpp-server.conference.example.org"},
		crtData: `-----BEGIN CERTIFICATE-----
MIIE2zCCA8OgAwIBAgIJAPpUBVr/QTXIMA0GCSqGSIb3DQEBBQUAMIGRMRQwEgYD
VQQDDAtleGFtcGxlLm9yZzELMAkGA1UEBhMCR0IxFTATBgNVBAcMDFRoZSBJbnRl
cm5ldDEaMBgGA1UECgwRWW91ciBPcmdhbmlzYXRpb24xGDAWBgNVBAsMD1hNUFAg
RGVwYXJ0bWVudDEfMB0GCSqGSIb3DQEJARYQeG1wcEBleGFtcGxlLm9yZzAeFw0x
NzA5MDcyMjMxMzFaFw0xODA5MDcyMjMxMzFaMIGRMRQwEgYDVQQDDAtleGFtcGxl
Lm9yZzELMAkGA1UEBhMCR0IxFTATBgNVBAcMDFRoZSBJbnRlcm5ldDEaMBgGA1UE
CgwRWW91ciBPcmdhbmlzYXRpb24xGDAWBgNVBAsMD1hNUFAgRGVwYXJ0bWVudDEf
MB0GCSqGSIb3DQEJARYQeG1wcEBleGFtcGxlLm9yZzCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBANb1dxZ7fCKy3CVrpvMhn59fi9BdnnSOPTM50NOtoVDF
gXqnGaIB+43t1gg/JIMudAcjH0go+HmI/Ljw1le8QdkY6zMxVkbG+lpYOOvz7PcS
HNsvhmA2ZW+fNHAV3DkHd+XsFi3KtrGbVQEOyZH+V+OC4DZv/Z6kwPzOu1OWvv0/
VRNiv0m4wzP0cz4j4nmCqgLOORN6OdlommiyABYk571Mk1ApfOOkMJmoOiJ1e7b0
jHaOANO++TZIKsvy6oXG6UHkcLoljAiplqEgtDEd/vdwnECEchJ4ttdkahX1OZw6
wuVXok8BnSjXH1y0FFdVDvb/AwRPpF7ZzMgYyfWRYRMCAwEAAaOCATIwggEuMAkG
A1UdEwQCMAAwCwYDVR0PBAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEF
BQcDAjCB9AYDVR0RBIHsMIHpggtleGFtcGxlLm9yZ6AZBggrBgEFBQcIBaANDAtl
eGFtcGxlLm9yZ6AmBggrBgEFBQcIB6AaFhhfeG1wcC1jbGllbnQuZXhhbXBsZS5v
cmegJgYIKwYBBQUHCAegGhYYX3htcHAtc2VydmVyLmV4YW1wbGUub3JnghZjb25m
ZXJlbmNlLmV4YW1wbGUub3JnoCQGCCsGAQUFBwgFoBgMFmNvbmZlcmVuY2UuZXhh
bXBsZS5vcmegMQYIKwYBBQUHCAegJRYjX3htcHAtc2VydmVyLmNvbmZlcmVuY2Uu
ZXhhbXBsZS5vcmcwDQYJKoZIhvcNAQEFBQADggEBAKJKOCiC4HPw/GiO+gBQTV4R
M1JJkYGY/knFvcrSxukH7yfk15AKNG4i0+6oMowV8OtYGFeXsSFXkteWVnLY7ULu
Yjlg2EZbandQw1eqfy25M3Oh/ZO+w4ZZ/yIzY2jMVayJv88kuS7BVBu6pL6Mm00X
0kmvLMvkb04IN3XYBYxsbzTscBOXUm7thGKzvGWSBAARnT24nxysyRbL0YE+gpB0
jEnlM0fUVuWWFdplC/2ktcQKUc6U47pXqpFrK2/3a28Ocn10hEw0uDVGX/UcMcRP
djT/rjumhQWaKI6yY3PEgz/GLdyiKZvH/0LsbYSQAKnxqS02EDwULcBTdSscbvM=
-----END CERTIFICATE-----`,
	},
}

func TestCertificates(t *testing.T) {
	for i, tc := range crtTests {
		blk, _ := pem.Decode([]byte(tc.crtData))
		t.Run(fmt.Sprintf("%d/Parse", i), func(t *testing.T) {
			crt, err := x509.ParseCertificate(blk.Bytes)
			switch {
			case err != tc.err:
				t.Fatal(err)
			case err != nil:
				return
			}
			doTests(t, tc, crt)
		})
		t.Run(fmt.Sprintf("%d/Cert", i), func(t *testing.T) {
			cryptocrt, err := cryptox509.ParseCertificate(blk.Bytes)
			switch {
			case err != tc.err:
				t.Error(err)
			case err != nil:
				return
			}
			crt, err := x509.FromCertificate(cryptocrt)
			switch {
			case err != tc.err:
				t.Fatal(err)
			case err != nil:
				return
			}
			doTests(t, tc, crt)
		})
	}
}

func doTests(t *testing.T, tc crtTest, crt *x509.Certificate) {
	if len(tc.srvNames) != len(crt.SRVNames) {
		t.Errorf("Wrong SRVNames want=%v, got=%v", tc.srvNames, crt.SRVNames)
	} else {
		for _, name := range tc.srvNames {
			found := false
			for _, otherName := range crt.SRVNames {
				if otherName == name {
					found = true
				}
			}
			if !found {
				t.Errorf("Wrong SRVNames want=%v, got=%v", tc.srvNames, crt.SRVNames)
				break
			}
		}
	}
	if len(tc.xmppAddrs) != len(crt.XMPPAddresses) {
		t.Errorf("Wrong XMPPAddrs want=%v, got=%v", tc.xmppAddrs, crt.XMPPAddresses)
	} else {
		for _, name := range tc.xmppAddrs {
			found := false
			for _, otherName := range crt.XMPPAddresses {
				if otherName == name {
					found = true
				}
			}
			if !found {
				t.Errorf("Wrong SRVNames want=%v, got=%v", tc.xmppAddrs, crt.XMPPAddresses)
				break
			}
		}
	}
}
