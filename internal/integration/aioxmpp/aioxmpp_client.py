# Copyright 2022 The Mellium Contributors.
# Use of this source code is governed by the BSD 2-clause
# license that can be found in the LICENSE file.

from aioxmpp import connector
from aioxmpp import JID
from aioxmpp import security_layer
import aioxmpp
import xmpptest

###
# This script acts as a wrapper around aioxmpp that starts a simple client.
# It is not meant to do any testing itself, instead use the Go python.Import
# option to create a test script that subclasses Daemon and overrides its run
# method.
###


class Daemon(xmpptest.Daemon):
    def __init__(self) -> None:
        super().__init__()

    def configure(self):
        super().configure()
        security: security_layer.SecurityLayer = aioxmpp.make_security_layer(
            self.config.get("client", "password"),
            no_verify=True,
        )
        self.jid: JID = JID.fromstr(self.config.get("client", "jid"))
        xmpp_port = self.config.getint("client", "port")
        conn: connector.BaseConnector = connector.STARTTLSConnector()
        self.client: aioxmpp.Client = aioxmpp.PresenceManagedClient(
            self.jid,
            security,
            override_peer=[("127.0.0.1", xmpp_port, conn)],
        )

    async def run_test(self) -> None:
        async with self.client.connected():
            await self.run()

    async def run(self) -> None:
        """
        The run method should be overridden by the test script and perform any
        actions required for testing such as listening for incoming messages and
        responding using self.client.
        """
        raise NotImplementedError("test file must override run function")
