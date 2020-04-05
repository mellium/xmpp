// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ejabberd

import (
	"path/filepath"
	"text/template"
)

// Config contains options that can be written to a Prosody config file.
type Config struct {
	VHosts []string
}

const inetrc = `{lookup,["file","native"]}.
{host,{127,0,0,1}, ["localhost","hostalias"]}.
{file, resolv, "/etc/resolv.conf"}.`

const cfgBase = `{{- if .VHosts }}
hosts:
{{- range .VHosts }}
  - {{.}}
{{- end }}

certfiles:
{{- range .VHosts }}
 - {{ filepathJoin $.ConfigDir . }}.crt
 - {{ filepathJoin $.ConfigDir . }}.key
{{- end }}
{{- end }}

loglevel: info

listen:
  -
    port: 5222
    ip: "::1"
    module: ejabberd_c2s
    max_stanza_size: 262144
    shaper: c2s_shaper
    access: c2s
    starttls_required: true

s2s_use_starttls: optional

modules:
  mod_adhoc: {}
  mod_admin_extra: {}
  mod_announce:
    access: announce
  mod_avatar: {}
  mod_blocking: {}
  mod_bosh: {}
  mod_caps: {}
  mod_carboncopy: {}
  mod_client_state: {}
  mod_configure: {}
  mod_disco: {}
  mod_fail2ban: {}
  mod_http_api: {}
  mod_http_upload:
    put_url: https://@HOST@:5443/upload
  mod_last: {}
  mod_mam:
    assume_mam_usage: true
    default: always
  mod_muc:
    access:
      - allow
    access_admin:
      - allow: admin
    access_create: muc_create
    access_persistent: muc_create
    access_mam:
      - allow
    default_room_options:
      mam: true
  mod_muc_admin: {}
  mod_offline:
    access_max_user_messages: max_user_offline_messages
  mod_ping: {}
  mod_privacy: {}
  mod_private: {}
  mod_pubsub:
    access_createnode: pubsub_createnode
    plugins:
      - flat
      - pep
    force_node_config:
      ## Avoid buggy clients to make their bookmarks public
      storage:bookmarks:
        access_model: whitelist
  mod_register:
    ip_access: trusted_network
  mod_roster:
    versioning: true
  mod_s2s_dialback: {}
  mod_shared_roster: {}
  mod_stream_mgmt:
    resend_on_timeout: if_offline
  mod_vcard: {}
  mod_vcard_xupdate: {}
  mod_version:
    show_os: false`

var cfgTmpl = template.Must(template.New("cfg").Funcs(template.FuncMap{
	"filepathJoin": filepath.Join,
}).Parse(cfgBase))
