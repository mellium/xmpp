-- Copyright 2022 The Mellium Contributors.
-- Use of this source code is governed by the BSD 2-clause
-- license that can be found in the LICENSE file.

module:depends("blocklist");

local storage = module:open_store();

local st = require"util.stanza";

module:add_feature("urn:mellium:integration");

module:hook("iq-set/self/urn:xmpp:blocking:block", function (event)
	for item in event.stanza.tags[1]:childtags("item") do
		local report = item:get_child("report", "urn:xmpp:reporting:1");
		if report then
			local text = report:get_child_text("text") or "";
			local jid = item.attr.jid;
			module:log("info", "--report: [%s, %s, %s]", jid, report.attr.reason, text);
			local reports = storage:get("reports") or {};
			reports[jid] = {
				jid = jid,
				reason = report.attr.reason,
				text = text,
			}
			storage:set("reports", reports)
		end
	end
end, 1);

module:hook("iq-get/self/urn:mellium:integration:report", function (event)
	local origin, stanza = event.origin, event.stanza;
	local reply = st.reply(stanza):tag("report", { xmlns = "urn:mellium:integration:report" });
	local reports = storage:get("reports")
	for jid, v in pairs(reports) do
		reply:tag("item", v):up();
	end
	origin.send(reply)
end, -1);
