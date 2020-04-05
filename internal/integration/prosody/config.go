// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package prosody

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

// Config contains options that can be written to a Prosody config file.
type Config struct {
	Admins []string
	VHosts []string
}

const cfgBase = `daemonize = false
pidfile = "{{ filepathJoin .ConfigDir "prosody.pid" }}"
admins = { {{ joinQuote .Admins }} }

modules_enabled = {

	-- Generally required
		"roster"; -- Allow users to have a roster. Recommended ;)
		"saslauth"; -- Authentication for clients and servers. Recommended if you want to log in.
		"tls"; -- Add support for secure TLS on c2s/s2s connections
		"dialback"; -- s2s dialback support
		"disco"; -- Service discovery

	-- Not essential, but recommended
		"carbons"; -- Keep multiple clients in sync
		"pep"; -- Enables users to publish their avatar, mood, activity, playing music and more
		"private"; -- Private XML storage (for room bookmarks, etc.)
		"blocklist"; -- Allow users to block communications with other users
		"vcard4"; -- User profiles (stored in PEP)
		"vcard_legacy"; -- Conversion between legacy vCard and PEP Avatar, vcard

	-- Nice to have
		"version"; -- Replies to server version requests
		"uptime"; -- Report how long server has been running
		"time"; -- Let others know the time here on this server
		"ping"; -- Replies to XMPP pings with pongs
		"register"; -- Allow users to register on this server using a client and change passwords

	-- Admin interfaces
		"admin_adhoc"; -- Allows administration via an XMPP client that supports ad-hoc commands
}

allow_registration = false
c2s_require_encryption = true
s2s_require_encryption = true
s2s_secure_auth = false
s2s_insecure_domains = { {{ joinQuote .VHosts }} }
authentication = "internal_hashed"
storage = "internal"

log = {
	"*console";
}

statistics = "internal"
certificates = "certs"

{{- range .VHosts }}
VirtualHost "{{ . }}"
{{- end }}`

var cfgTmpl = template.Must(template.New("cfg").Funcs(template.FuncMap{
	"filepathJoin": filepath.Join,
	"joinQuote": func(s []string) string {
		s = append(s[:0:0], s...)
		for i, ss := range s {
			s[i] = fmt.Sprintf("%q", ss)
		}
		return strings.Join(s, ",")
	},
}).Parse(cfgBase))
