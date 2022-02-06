# Copyright 2022 The Mellium Contributors.
# Use of this source code is governed by the BSD 2-clause
# license that can be found in the LICENSE file.

from slixmpp_client import Daemon
from xml.etree import cElementTree as ET
from asyncio import Future
import slixmpp


class SendIBB(Daemon):
    def __init__(self) -> None:
        super().__init__()
        self.data = bytearray()

    def prepare_argparse(self) -> None:
        super().prepare_argparse()

        self.argparse.add_argument(
            "-j",
            type=slixmpp.jid.JID,
            help="The JID to start an IBB session with",
        )

    def configure(self) -> None:
        super().configure()
        self.client.register_plugin('xep_0047')

    async def run(self) -> None:
        ibb = self.client.plugin['xep_0047']

        self.client.add_event_handler(
            "ibb_stream_data",
            lambda stream: self.data.extend(stream.recv_queue.get_nowait()),
        )

        conn = await ibb.open_stream(self.args.j)
        await conn.sendall("Warren snores through the night like a bear—a bass to the treble of the loons.".encode('utf-8'))
        await conn.close()

        # Echo the data we read back so that the other side can confirm that
        # what it sent is what was received.
        msg = slixmpp.stanza.Message()
        msg = self.client.make_message(
            mto=self.args.j,
            mbody=self.data.decode('utf-8'),
        )
        msg.appendxml(ET.Element('doneibb'))
        msg.send()


class RecvIBB(Daemon):
    def __init__(self) -> None:
        super().__init__()
        self.data = bytearray()
        self.conn = Future()

    def prepare_argparse(self) -> None:
        super().prepare_argparse()

        self.argparse.add_argument(
            "-j",
            type=slixmpp.jid.JID,
            help="The JID to expect an IBB session from",
        )
        self.argparse.add_argument(
            "-sid",
            type=str,
            help="The IBB session ID to expect",
        )

    def configure(self) -> None:
        super().configure()
        self.end = Future()
        self.client.register_plugin('xep_0047', {
            'auto_accept': True,
        })
        self.client.add_event_handler(
            "ibb_stream_data",
            lambda stream: self.data.extend(stream.recv_queue.get_nowait()),
        )
        self.client.add_event_handler(
            "ibb_stream_end",
            lambda stream: self.end.set_result(True)
        )
        self.client.add_event_handler("ibb_stream_start", lambda conn: self.conn.set_result(conn))

    async def run(self) -> None:
        ibb = self.client.plugin['xep_0047']

        # First send the other side of the connection our JID so that it knows
        # what JID to use.
        msg = slixmpp.stanza.Message()
        msg = self.client.make_message(
            mto=self.args.j,
            mbody=self.data.decode('utf-8'),
        )
        msg.appendxml(ET.Element('startibb'))
        msg.send()

        conn = await self.conn
        await conn.sendall(b"I feel a deep security in the single-mindedness of freight trains.")
        await self.end

        # Echo the data we read back so that the other side can confirm that
        # what it sent is what was received.
        msg = slixmpp.stanza.Message()
        msg = self.client.make_message(
            mto=self.args.j,
            mbody=self.data.decode('utf-8'),
        )
        msg.appendxml(ET.Element('doneibb'))
        msg.send()
