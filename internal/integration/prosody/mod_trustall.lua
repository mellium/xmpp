-- Copyright 2022 The Mellium Contributors.
-- Use of this source code is governed by the BSD 2-clause
-- license that can be found in the LICENSE file.

module:set_global();

module:hook("s2s-check-certificate", function(event)
	local session = event.session;
	module:log("info", "implicitly trusting presented certificate");
	session.cert_chain_status = "valid";
	session.cert_identity_status = "valid";
	return true;
end);
