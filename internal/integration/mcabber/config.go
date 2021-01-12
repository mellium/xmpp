// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package mcabber

import (
	"os"
	"path/filepath"
	"text/template"

	"mellium.im/xmpp/jid"
)

// Config contains options that can be written to the config file.
type Config struct {
	JID      jid.JID
	Password string
	FIFO     *os.File
	Port     string
}

const cfgBase = `set jid = {{.JID}}
set vi_mode=1
set password = {{.Password}}
set server = localhost
set port = {{ .Port }}
set resource = mcabbertest
set disable_random_resource = 0
set tls = 1
set ssl_ignore_checks = 1
set pinginterval = 0
set spell_enable = 0
set disable_chatstates = 1
set logging = 1
set load_logs = 0
set logging_dir = {{ .ConfigDir }}
set log_muc_conf = 1
set statefile = {{ filepathJoin .ConfigDir "mcabber.state" }}


# Modules
# If mcabber is built with modules support, you can specify the path
# to the directory where your modules reside. Though, default compiled-in
# value should be appropriate.
#set modules_dir = /usr/lib/mcabber/

set beep_on_message = 0
set event_log_files = 1
set event_log_dir = {{ .ConfigDir }}

# Internal hooks
# You can ask mcabber to execute an internal command when a special event
# occurs (for example when it connects to the server).
#
# 'hook-post-connect' is executed when mcabber has connected to the server
# and the roster has been received.
#set hook-post-connect = status dnd
#
# 'hook-pre-disconnect' is executed just before mcabber disconnects from
# the server.
#set hook-pre-disconnect = say_to foo@bar Goodbye!

# FIFO
# mcabber can create a FIFO named pipe and listen to this pipe for commands.
# Don't forget to load the FIFO module if you plan to use this feature!
# Default: disabled.
# Set 'fifo_hide_commands' to 1 if you don't want to see the FIFO commands
# in the log window (they will still be written to the tracelog file).
# When FIFO  is configured, you can turn it off and on in real time with
# the 'fifo_ignore' option (default: 0).  When set to 1, the FIFO input is
# still read but it is discarded.
set fifo_name = {{ .FIFO.Name }}
module load fifo


# Traces logging
# If you want advanced traces, please specify a file and a level here.
# There are currently 4 tracelog levels:
#  lvl 1: most events of the log window are written to the file
#  lvl 2: Loudmouth verbose logging
#  lvl 3: debug logging (XML, etc.)
#  lvl 4: noisy debug logging (Loudmouth parser...)
# Default is level 0, no trace logging
set tracelog_level = 1
set tracelog_file = {{ filepathJoin .ConfigDir "trace.fifo" }}
set autoaway = 0
set muc_print_status = 3
set muc_print_jid = 2
set log_display_presence = 1
set show_status_in_buffer = 2
set log_display_sender = 1
set caps_directory = "{{ .ConfigDir }}"
`

var cfgTmpl = template.Must(template.New("cfg").Funcs(template.FuncMap{
	"filepathJoin": filepath.Join,
}).Parse(cfgBase))
