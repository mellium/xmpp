# Copyright 2022 The Mellium Contributors.
# Use of this source code is governed by the BSD 2-clause
# license that can be found in the LICENSE file.

from slixmpp import jid
import slixmpp
import ssl
import xmpptest


###
# This script acts as a wrapper around slixmpp that starts a simple client.
# It is not meant to do any testing itself, instead use the Go python.Import
# option to create a test script that subclasses Daemon and overrides its run
# method.
###

class Daemon(xmpptest.Daemon):
    def __init__(self) -> None:
        super().__init__()

    def configure(self):
        super().configure()
        self.jid: jid.JID = jid.JID(self.config.get("client", "jid"))
        self.xmpp_port = self.config.getint("client", "port")
        self.client: slixmpp.ClientXMPP = slixmpp.ClientXMPP(
            self.jid,
            self.config.get("client", "password"),
        )

    async def run_test(self) -> None:
        async def run_callback(_):
            await self.run()
        self.client.add_event_handler('session_start', run_callback)
        # These are integration tests that don't use a real certificate
        # so disable verification so that our self-signed certs work.
        tlsCtx = self.client.get_ssl_context()
        tlsCtx.check_hostname=False
        tlsCtx.verify_mode=ssl.CERT_NONE
        self.client.connect(address=("127.0.0.1", self.xmpp_port),
                            force_starttls=False)
        await self.client.wait_until('session_end')

    async def run(self) -> None:
        """
        The run method should be overridden by the test script and perform any
        actions required for testing such as listening for incoming messages and
        responding using self.client.
        """
        raise NotImplementedError("test file must override run function")
