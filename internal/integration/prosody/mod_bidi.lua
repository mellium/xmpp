-- Lua code in this file was copied from the Prosody community modules and is
-- used under the terms of the MIT License, a copy of which can be found in the
-- file LICENSE.modules.
-- The code may have been modified and no longer matches upstream.
--
-- https://modules.prosody.im/mod_bidi
--
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
