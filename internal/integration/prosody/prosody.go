// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.
//
// Some Lua embedded in this file is taken from the Prosody Community Modules
// and is licensed under the terms of the MIT license, a copy of which can be
// found in the file "LICENSE.modules".

// Package prosody facilitates integration testing against Prosody.
package prosody // import "mellium.im/xmpp/internal/integration/prosody"

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"testing"

	"mellium.im/xmpp/internal/integration"
	"mellium.im/xmpp/jid"
)

const (
	cfgFileName = "prosody.cfg.lua"
	cmdName     = "prosody"
	configFlag  = "--config"
)

// New creates a new, unstarted, prosody daemon.
//
// The provided context is used to kill the process (by calling os.Process.Kill)
// if the context becomes done before the command completes on its own.
func New(ctx context.Context, opts ...integration.Option) (*integration.Cmd, error) {
	return integration.New(
		ctx, cmdName,
		opts...,
	)
}

// WebSocket enables the websocket module.
// WebSocket implies the HTTPS() option.
func WebSocket() integration.Option {
	return func(cmd *integration.Cmd) error {
		err := Modules("websocket")(cmd)
		if err != nil {
			return err
		}
		return HTTPS()(cmd)
	}
}

// ConfigFile is an option that can be used to write a temporary Prosody config
// file.
// This will overwrite the existing config file and make most of the other
// options in this package noops.
// This option only exists for the rare occasion that you need complete control
// over the config file.
func ConfigFile(cfg Config) integration.Option {
	return func(cmd *integration.Cmd) error {
		cmd.Config = cfg
		err := integration.TempFile(cfgFileName, func(cmd *integration.Cmd, w io.Writer) error {
			return cfgTmpl.Execute(w, struct {
				Config
				ConfigDir string
			}{
				Config:    cfg,
				ConfigDir: cmd.ConfigDir(),
			})
		})(cmd)
		if err != nil {
			return err
		}
		cfgFilePath := filepath.Join(cmd.ConfigDir(), cfgFileName)
		return integration.Args(configFlag, cfgFilePath)(cmd)
	}
}

// Ctl returns an option that calls prosodyctl with the provided args.
// It automatically points prosodyctl at the config file so there is no need to
// pass the --config option.
func Ctl(ctx context.Context, args ...string) integration.Option {
	return integration.Defer(ctlFunc(ctx, args...))
}

func ctlFunc(ctx context.Context, args ...string) func(*integration.Cmd) error {
	return func(cmd *integration.Cmd) error {
		cfgFilePath := filepath.Join(cmd.ConfigDir(), cfgFileName)
		/* #nosec */
		prosodyCtl := exec.CommandContext(ctx, "prosodyctl", configFlag, cfgFilePath)
		prosodyCtl.Args = append(prosodyCtl.Args, args...)
		return prosodyCtl.Run()
	}
}

func getConfig(cmd *integration.Cmd) Config {
	if cmd.Config == nil {
		cmd.Config = Config{}
	}
	return cmd.Config.(Config)
}

// ListenC2S listens for client-to-server (c2s) connections on a random port.
func ListenC2S() integration.Option {
	return func(cmd *integration.Cmd) error {
		c2sListener, err := cmd.C2SListen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		// Prosody creates its own sockets and doesn't provide us with a way of
		// pointing it at an existing Unix domain socket or handing the filehandle
		// for the TCP connection to it on start, so we're effectively just
		// listening to get a random port that we'll use to configure Prosody, then
		// we need to close the connection and let Prosody listen on that port.
		// Technically this is racey, but it's not likely to be a problem in
		// practice.
		c2sPort := c2sListener.Addr().(*net.TCPAddr).Port
		err = c2sListener.Close()
		if err != nil {
			return err
		}

		cfg := getConfig(cmd)
		cfg.C2SPort = c2sPort
		cmd.Config = cfg
		return nil
	}
}

// ListenS2S listens for server-to-server (s2s) connections on a random port.
func ListenS2S() integration.Option {
	return func(cmd *integration.Cmd) error {
		s2sListener, err := cmd.S2SListen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		// Prosody creates its own sockets and doesn't provide us with a way of
		// pointing it at an existing Unix domain socket or handing the filehandle for
		// the TCP connection to it on start, so we're effectively just listening to
		// get a random port that we'll use to configure Prosody, then we need to
		// close the connection and let Prosody listen on that port.
		// Technically this is racey, but it's not likely to be a problem in practice.
		s2sPort := s2sListener.Addr().(*net.TCPAddr).Port
		err = s2sListener.Close()
		if err != nil {
			return err
		}

		cfg := getConfig(cmd)
		cfg.S2SPort = s2sPort
		cmd.Config = cfg
		return nil
	}
}

// VHost configures one or more virtual hosts.
// The default if this option is not provided is to create a single vhost called
// "localhost" and create a self-signed cert for it (if VHost is specified certs
// must be manually created).
func VHost(hosts ...string) integration.Option {
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		cfg.VHosts = append(cfg.VHosts, hosts...)
		cmd.Config = cfg
		return nil
	}
}

// MUC launches prosody with the built-in multi-user chat component enabled.
// It is the same as Component(domain, "", "muc", modules).
func MUC(domain string, modules ...string) integration.Option {
	return Component(domain, "", "muc", modules...)
}

// Channel configures the MUC component (if loaded) with a default channel or
// channels.
func Channel(domain string, c ...ChannelConfig) integration.Option {
	const modName = "muc_defaults"
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		comp := cfg.Component[domain]
		comp.MUCDefaults = append(comp.MUCDefaults, c...)
		comp.Modules = append(comp.Modules, modName)
		cfg.Component[domain] = comp
		cmd.Config = cfg
		return integration.TempFile("mod_"+modName+".lua", func(_ *integration.Cmd, w io.Writer) error {
			_, err := io.WriteString(w, `local log = module._log;
local params = module:get_option("default_mucs", {});
local jid_bare = require "util.jid".bare;


local function set_affiliations(room, affiliations)
	for affiliation, jids in pairs(affiliations) do
		for i, jid in pairs(jids) do
			module:log("debug", "Setting affiliation %s for jid %s", affiliation, jid);
			room:set_affiliation(true, jid_bare(jid), affiliation);
		end
	end
end


local function configure_room(room, config)
	local should_save = false;
	if config.name ~= nil then
		should_save = room:set_name(config.name) or should_save;
	end
	if config.description ~= nil then
		should_save = room:set_description(config.description) or should_save;
	end
	if config.allow_member_invites ~= nil then
		should_save =
			room:set_allow_member_invites(config.allow_member_invites)
			or should_save;
	end
	if config.change_subject ~= nil then
		should_save =
			room:set_changesubject(config.change_subject)
			or should_save;
	end
	if config.history_length ~= nil then
		should_save =
			room:set_historylength(config.history_length)
			or should_save;
	end
	if config.lang ~= nil then
		should_save = room:set_language(config.lang) or should_save;
	end
	if config.pass ~= nil then
		should_save = room:set_password(config.pass) or should_save;
	end
	if config.members_only ~= nil then
		should_save =
			room:set_members_only(config.members_only)
			or should_save;
	end
	if config.moderated ~= nil then
		should_save = room:set_moderated(config.moderated) or should_save;
	end
	if config.persistent ~= nil then
		should_save = room:set_persistent(config.persistent) or should_save;
	end
	if config.presence_broadcast ~= nil then
		should_save = room:set_presence_broadcast(config.presence_broadcast) or should_save;
	end
	if config.public ~= nil then
		should_save = room:set_hidden(not config.public) or should_save;
	end
	if config.public_jids ~= nil then
		should_save =
			room:set_whois(config.public_jids and "anyone" or "moderators")
			or should_save;
	end
	if config.logging ~= room._data.logging then
		room._data.logging = config.logging;
		should_save = true;
	end
	if should_save then
		room:save(true);
	end
end


local i, room_data;
for i, room_data in pairs(params) do
	local host = module.host;
	local room_jid = room_data.jid_node.."@"..host;
	local mod_muc = prosody.hosts[host].modules.muc;
	local room = mod_muc.get_room_from_jid(room_jid);
	if not room then
		module:log("debug", "Creating new room %s", room_jid);
		-- We don't pass in the config, so that the default config is set first.
		room = mod_muc.create_room(room_jid);
	else
		module:log("debug", "Configuring already existing room %s", room_jid);
	end
	configure_room(room, room_data.config);
	if room_data.affiliations then
		set_affiliations(room, room_data.affiliations);
	end
end`)
			return err
		})(cmd)
	}
}

// Component adds an component with the given domain and secret to the config
// file.
// If a name is provided the component must be a builtin.
func Component(domain, secret, name string, modules ...string) integration.Option {
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		if name == "" {
			compListener, err := cmd.ComponentListen("tcp", "127.0.0.1:0")
			if err != nil {
				return err
			}
			// Prosody creates its own sockets and doesn't provide us with a way of
			// pointing it at an existing Unix domain socket or handing the filehandle
			// for the TCP connection to it on start, so we're effectively just
			// listening to get a random port that we'll use to configure Prosody, then
			// we need to close the connection and let Prosody listen on that port.
			// Technically this is racey, but it's not likely to be a problem in
			// practice.
			compPort := compListener.Addr().(*net.TCPAddr).Port
			err = compListener.Close()
			if err != nil {
				return err
			}

			cfg.CompPort = compPort
		}
		if cfg.Component == nil {
			cfg.Component = make(map[string]struct {
				Name        string
				Secret      string
				Modules     []string
				MUCDefaults []ChannelConfig
			})
		}
		comp := cfg.Component[domain]
		comp.Secret = secret
		comp.Name = name
		comp.Modules = modules
		cfg.Component[domain] = comp
		cmd.Config = cfg
		return nil
	}
}

// HTTPS configures prosody to listen for HTTP and HTTPS on two randomized
// ports and configures TLS certificates for localhost:https.
func HTTPS() integration.Option {
	return func(cmd *integration.Cmd) error {
		httpsListener, err := cmd.HTTPSListen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		httpListener, err := cmd.HTTPListen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}

		// Prosody creates its own sockets and doesn't provide us with a way of
		// pointing it at an existing Unix domain socket or handing the filehandle
		// for the TCP connection to it on start, so we're effectively just
		// listening to get a random port that we'll use to configure Prosody, then
		// we need to close the connection and let Prosody listen on that port.
		// Technically this is racey, but it's not likely to be a problem in
		// practice.
		httpPort := httpListener.Addr().(*net.TCPAddr).Port
		httpsPort := httpsListener.Addr().(*net.TCPAddr).Port
		err = httpListener.Close()
		if err != nil {
			return err
		}
		err = httpsListener.Close()
		if err != nil {
			return err
		}

		cfg := getConfig(cmd)
		cfg.HTTPPort = httpPort
		cfg.HTTPSPort = httpsPort
		cmd.Config = cfg
		return integration.Cert(fmt.Sprintf("localhost:%d", httpsPort))(cmd)
	}
}

// CreateUser returns an option that calls prosodyctl to create a user.
// It is equivalent to calling:
// Ctl(ctx, "register", "localpart", "domainpart", "password") except that it
// also configures the underlying Cmd to know about the user.
func CreateUser(ctx context.Context, addr, pass string) integration.Option {
	return func(cmd *integration.Cmd) error {
		j, err := jid.Parse(addr)
		if err != nil {
			return err
		}
		err = Ctl(ctx, "register", j.Localpart(), j.Domainpart(), pass)(cmd)
		if err != nil {
			return err
		}
		return integration.User(j, pass)(cmd)
	}
}

// Modules adds custom modules to the enabled modules list.
func Modules(mod ...string) integration.Option {
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		cfg.Modules = append(cfg.Modules, mod...)
		cmd.Config = cfg
		return nil
	}
}

// Set adds an extra key/value pair to the global section of the config file.
// If v is a string it will be quoted, otherwise it is marshaled using the %v
// formatting directive (see the fmt package for details).
// As a special case, if v is nil the key is written to the file directly with
// no equals sign.
//
//     -- Set("foo", "bar")
//     foo = "bar"
//
//     -- Set("foo", 123)
//     foo = 123
//
//     -- Set(`Component "conference.example.org" "muc"`, nil)
//     Component "conference.example.org" "muc"
func Set(key string, v interface{}) integration.Option {
	return func(cmd *integration.Cmd) error {
		cfg := getConfig(cmd)
		if cfg.Options == nil {
			cfg.Options = make(map[string]interface{})
		}
		cfg.Options[key] = v
		cmd.Config = cfg
		return nil
	}
}

// Bidi enables bidirectional S2S connections.
func Bidi() integration.Option {
	// TODO: Once Prosody 0.12 is out this module can be replaced with the builtin
	// mod_s2s_bidi. See https://mellium.im/issue/78
	const modName = "bidi"
	return func(cmd *integration.Cmd) error {
		err := Modules(modName)(cmd)
		if err != nil {
			return err
		}
		return integration.TempFile("mod_"+modName+".lua", func(_ *integration.Cmd, w io.Writer) error {
			_, err := io.WriteString(w, `
-- Bidirectional Server-to-Server Connections
-- http://xmpp.org/extensions/xep-0288.html
-- Copyright (C) 2013 Kim Alvefur
--
-- This file is MIT/X11 licensed.
--
local add_filter = require "util.filters".add_filter;
local st = require "util.stanza";
local jid_split = require"util.jid".prepped_split;
local core_process_stanza = prosody.core_process_stanza;
local traceback = debug.traceback;
local hosts = hosts;
local xmlns_bidi_feature = "urn:xmpp:features:bidi"
local xmlns_bidi = "urn:xmpp:bidi";
local secure_only = module:get_option_boolean("secure_bidi_only", true);
local disable_bidi_for = module:get_option_set("no_bidi_with", { });
local bidi_sessions = module:shared"sessions-cache";

local function handleerr(err) log("error", "Traceback[s2s]: %s: %s", tostring(err), traceback()); end
local function handlestanza(session, stanza)
	if stanza.attr.xmlns == "jabber:client" then --COMPAT: Prosody pre-0.6.2 may send jabber:client
		stanza.attr.xmlns = nil;
	end
	-- stanza = session.filter("stanzas/in", stanza);
	if stanza then
		return xpcall(function () return core_process_stanza(session, stanza) end, handleerr);
	end
end

local function new_bidi(origin)
	if origin.type == "s2sin" then -- then we create an "outgoing" bidirectional session
		local conflicting_session = hosts[origin.to_host].s2sout[origin.from_host]
		if conflicting_session then
			conflicting_session.log("info", "We already have an outgoing connection to %s, closing it...", origin.from_host);
			conflicting_session:close{ condition = "conflict", text = "Replaced by bidirectional stream" }
		end
		bidi_sessions[origin.from_host] = origin;
		origin.is_bidi = true;
		origin.outgoing = true;
	elseif origin.type == "s2sout" then -- handle incoming stanzas correctly
		local bidi_session = {
			type = "s2sin"; direction = "incoming";
			incoming = true;
			is_bidi = true; orig_session = origin;
			to_host = origin.from_host;
			from_host = origin.to_host;
			hosts = {};
		}
		origin.bidi_session = bidi_session;
		setmetatable(bidi_session, { __index = origin });
		module:fire_event("s2s-authenticated", { session = bidi_session, host = origin.to_host });
		local remote_host = origin.to_host;
		add_filter(origin, "stanzas/in", function(stanza)
			if stanza.attr.xmlns ~= nil then return stanza end
			local _, host = jid_split(stanza.attr.from);
			if host ~= remote_host then return stanza end
			handlestanza(bidi_session, stanza);
		end, 1);
	end
end

module:hook("route/remote", function(event)
	local from_host, to_host, stanza = event.from_host, event.to_host, event.stanza;
	if from_host ~= module.host then return end
	local to_session = bidi_sessions[to_host];
	if not to_session or to_session.type ~= "s2sin" then return end
	if to_session.sends2s(stanza) then return true end
end, -2);

-- Incoming s2s
module:hook("s2s-stream-features", function(event)
	local origin, features = event.origin, event.features;
	if not origin.is_bidi and not origin.bidi_session and not origin.do_bidi
	and not hosts[module.host].s2sout[origin.from_host]
	and not disable_bidi_for:contains(origin.from_host)
	and (not secure_only or (origin.cert_chain_status == "valid"
	and origin.cert_identity_status == "valid")) then
		if origin.incoming == true then
			module:log("warn", "This module can now be replaced by mod_s2s_bidi which is included with Prosody");
		end
		module:log("debug", "Announcing support for bidirectional streams");
		features:tag("bidi", { xmlns = xmlns_bidi_feature }):up();
	end
end);

module:hook("stanza/urn:xmpp:bidi:bidi", function(event)
	local origin = event.session or event.origin;
	if not origin.is_bidi and not origin.bidi_session
	and not disable_bidi_for:contains(origin.from_host)
	and (not secure_only or origin.cert_chain_status == "valid"
	and origin.cert_identity_status == "valid") then
		module:log("debug", "%s requested bidirectional stream", origin.from_host);
		origin.do_bidi = true;
		return true;
	end
end);

-- Outgoing s2s
module:hook("stanza/http://etherx.jabber.org/streams:features", function(event)
	local origin = event.session or event.origin;
	if not ( origin.bidi_session or origin.is_bidi or origin.do_bidi)
	and not disable_bidi_for:contains(origin.to_host)
	and event.stanza:get_child("bidi", xmlns_bidi_feature)
	and (not secure_only or origin.cert_chain_status == "valid"
	and origin.cert_identity_status == "valid") then
		if origin.outgoing == true then
			module:log("warn", "This module can now be replaced by mod_s2s_bidi which is included with Prosody");
		end
		module:log("debug", "%s supports bidirectional streams", origin.to_host);
		origin.sends2s(st.stanza("bidi", { xmlns = xmlns_bidi }));
		origin.do_bidi = true;
	end
end, 160);

function enable_bidi(event)
	local session = event.session;
	if session.do_bidi and not ( session.is_bidi or session.bidi_session ) then
		session.do_bidi = nil;
		new_bidi(session);
	end
end

module:hook("s2sin-established", enable_bidi);
module:hook("s2sout-established", enable_bidi);

function disable_bidi(event)
	local session = event.session;
	if session.type == "s2sin" then
		bidi_sessions[session.from_host] = nil;
	end
end

module:hook("s2sin-destroyed", disable_bidi);
module:hook("s2sout-destroyed", disable_bidi);
`)
			return err
		})(cmd)
	}
}

// TrustAll configures prosody to trust all certificates presented to it without
// any verification.
func TrustAll() integration.Option {
	const modName = "trustall"
	return func(cmd *integration.Cmd) error {
		err := Modules(modName)(cmd)
		if err != nil {
			return err
		}
		return integration.TempFile("mod_"+modName+".lua", func(_ *integration.Cmd, w io.Writer) error {
			_, err := io.WriteString(w, `
module:set_global();

module:hook("s2s-check-certificate", function(event)
	local session = event.session;
	module:log("info", "implicitly trusting presented certificate");
	session.cert_chain_status = "valid";
	session.cert_identity_status = "valid";
	return true;
end);`)
			return err
		})(cmd)
	}
}

func defaultConfig(cmd *integration.Cmd) error {
	for _, arg := range cmd.Cmd.Args {
		if arg == configFlag {
			return nil
		}
	}

	cfg := getConfig(cmd)
	if len(cfg.VHosts) == 0 {
		const vhost = "localhost"
		cfg.VHosts = append(cfg.VHosts, vhost)
		err := integration.Cert(vhost)(cmd)
		if err != nil {
			return err
		}
	}
	cmd.Config = cfg
	if j, _ := cmd.User(); j.Equal(jid.JID{}) {
		err := CreateUser(context.TODO(), "me@"+cfg.VHosts[0], "password")(cmd)
		if err != nil {
			return err
		}
	}

	return ConfigFile(cfg)(cmd)
}

// Test starts a Prosody instance and returns a function that runs subtests
// using t.Run.
// Multiple calls to the returned function will result in uniquely named
// subtests.
// When all subtests have completed, the daemon is stopped.
func Test(ctx context.Context, t *testing.T, opts ...integration.Option) integration.SubtestRunner {
	opts = append(opts, defaultConfig,
		integration.Shutdown(ctlFunc(ctx, "stop")))
	return integration.Test(ctx, cmdName, t, opts...)
}
