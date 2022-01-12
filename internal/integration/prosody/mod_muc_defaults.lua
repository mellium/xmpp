-- Lua code in this file was copied from the Prosody community modules and is
-- used under the terms of the MIT License, a copy of which can be found in the
-- file LICENSE.modules.
-- The code may have been modified and no longer matches upstream.
--
-- https://modules.prosody.im/mod_muc_defaults

local log = module._log;
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
end
