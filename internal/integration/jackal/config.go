// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package jackal

import (
	_ "embed"
	"path/filepath"
	"text/template"
)

// Config contains options that can be written to a Jackal config file.
type Config struct {
	C2SPort   int
	AdminPort int
	HTTPPort  int
	VHosts    []string
	Modules   []string
}

var (
	//go:embed config.yml.tmpl
	cfgBase string

	cfgTmpl = template.Must(template.New("cfg").Funcs(template.FuncMap{
		"filepathJoin": filepath.Join,
	}).Parse(cfgBase))
)
