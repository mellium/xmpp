// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package sendxmpp

import (
	"text/template"

	"mellium.im/xmpp/jid"
)

// Config contains options that can be written to a sendxmpp config file.
type Config struct {
	JID       jid.JID
	Port      string
	Password  string
	Component string
}

const cfgBase = `username: {{ .JID.Localpart }}
jserver: {{ .JID.Domainpart }}
port: {{ .Port }}
password: {{ .Password }}
{{ if .Component}}component: {{ .Component }}{{ end }}`

var cfgTmpl = template.Must(template.New("cfg").Parse(cfgBase))
