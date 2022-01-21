// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package prosody

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

// Config contains options that can be written to a Prosody config file.
type Config struct {
	C2SPort   int
	S2SPort   int
	CompPort  int
	HTTPPort  int
	HTTPSPort int
	Admins    []string
	Modules   []string
	VHosts    []string
	Options   map[string]interface{}
	Component map[string]struct {
		Name        string
		Secret      string
		Modules     []string
		MUCDefaults []ChannelConfig
	}
}

// ChannelConfig configures a Multi-User Chat channel.
type ChannelConfig struct {
	Localpart          string
	Admins             []string
	Owners             []string
	Visitors           []string
	Name               string
	Desc               string
	AllowMemberInvites bool
	ChangeSubject      bool
	HistoryLen         int
	Lang               string
	Pass               string
	Logging            bool
	MembersOnly        bool
	Moderated          bool
	Persistent         bool
	Public             bool
	PublicJIDs         bool
}

var (
	//go:embed prosody.cfg.lua.tmpl
	cfgBase string

	cfgTmpl = template.Must(template.New("cfg").Funcs(template.FuncMap{
		"filepathJoin": filepath.Join,
		"joinQuote": func(s []string) string {
			s = append(s[:0:0], s...)
			for i, ss := range s {
				s[i] = fmt.Sprintf("%q", ss)
			}
			return strings.Join(s, ",")
		},
		"luaList": func(s []string) string {
			s = append(s[:0:0], s...)
			for i, ss := range s {
				s[i] = fmt.Sprintf("%q", ss)
			}
			var end string
			if len(s) > 0 {
				end = ";\n"
			}
			return strings.Join(s, ";\n") + end
		},
		"quoteOrPrint": func(v interface{}) string {
			switch vv := v.(type) {
			case string:
				return fmt.Sprintf("%q", vv)
			default:
				return fmt.Sprintf("%v", vv)
			}
		},
	}).Parse(cfgBase))
)
