# Copyright 2022 The Mellium Contributors.
# Use of this source code is governed by the BSD 2-clause
# license that can be found in the LICENSE file.

from slixmpp_client import Daemon
import slixmpp


class Ping(Daemon):
    def prepare_argparse(self) -> None:
        super().prepare_argparse()

        self.argparse.add_argument(
            "-j",
            type=slixmpp.jid.JID,
            help="The JID to ping",
        )

    def configure(self):
        super().configure()
        self.client.register_plugin('xep_0199') # Ping

    async def run(self) -> None:
        await self.client.plugin['xep_0199'].ping(jid=self.args.j)
