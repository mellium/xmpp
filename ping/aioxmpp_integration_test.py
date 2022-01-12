# Copyright 2022 The Mellium Contributors.
# Use of this source code is governed by the BSD 2-clause
# license that can be found in the LICENSE file.

from aioxmpp_client import Daemon
import aioxmpp


class Ping(Daemon):
    def prepare_argparse(self) -> None:
        super().prepare_argparse()

        def jid(s):
            return aioxmpp.JID.fromstr(s)
        self.argparse.add_argument(
            "-j",
            type=jid,
            help="The JID to ping",
        )

    async def run(self) -> None:
        await aioxmpp.ping.ping(self.client, self.args.j)
