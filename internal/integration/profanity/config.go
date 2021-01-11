// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package profanity

import (
	"path/filepath"
	"text/template"

	"mellium.im/xmpp/jid"
)

// Config contains options that can be written to the config file.
type Config struct {
	JID      jid.JID
	Password string
	Port     string
}

const cfgAccount = `[testing]
enabled=true
jid={{.JID}}
resource=profanity.D0gD
muc.nick=profanity
presence.last=online
presence.login=online
priority.online=0
priority.chat=0
priority.away=0
priority.xa=0
priority.dnd=0
port={{.Port}}
server=127.0.0.1
password={{.Password}}
tls.policy=trust
`

const cfgBase = `[connection]
account=testing

[chatstates]
enabled=false

[notifications]
invite=false
sub=false
message=false
room=false
message.current=false
room.current=false
typing=false
typing.current=false
message.text=false
room.text=false

[logging]
rotate=false`

var cfgTmpl = template.Must(template.New("cfg").Funcs(template.FuncMap{
	"filepathJoin": filepath.Join,
}).Parse(cfgBase))

var accountsTmpl = template.Must(template.New("cfg").Funcs(template.FuncMap{
	"filepathJoin": filepath.Join,
}).Parse(cfgAccount))
