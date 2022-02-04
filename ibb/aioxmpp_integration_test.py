# Copyright 2022 The Mellium Contributors.
# Use of this source code is governed by the BSD 2-clause
# license that can be found in the LICENSE file.

import asyncio
import sys

from aioxmpp import ibb, JID
from aioxmpp import Message
from aioxmpp import MessageType
from aioxmpp import xso
from aioxmpp.utils import namespaces

import aioxmpp_client


class TestProtocol(asyncio.Protocol):

    def __init__(self) -> None:
        self.data = b""
        self.transport = None
        self.closed_fut = asyncio.Future()

    def connection_made(self, transport):
        self.transport = transport

    def connection_lost(self, e):
        self.closed_fut.set_result(e)

    def pause_writing(self):
        pass

    def resume_writing(self):
        pass

    def data_received(self, data):
        self.data += data


class StartIBB(xso.XSO):
    """
    When we're ready for the Go side to send an IBB open request we send
    <startibb/> as a way to give the Go side the JID we negotiated and let it
    know that we're ready to receive it.
    """
    TAG = (namespaces.client, "startibb")


class DoneIBB(xso.XSO):
    """
    When we finish receiving data over IBB we send it to the Go side via a
    message with the <doneibb/> payload and the data in the <body/>. This way
    the Go side can do the comparison and see if it matches what was expected.
    """
    TAG = (namespaces.client, "doneibb")


class SendIBB(aioxmpp_client.Daemon):
    def __init__(self) -> None:
        super().__init__()

    def prepare_argparse(self) -> None:
        super().prepare_argparse()

        def jid(s):
            return JID.fromstr(s)
        self.argparse.add_argument(
            "-j",
            type=jid,
            help="The JID to start an IBB session with",
        )

    async def run(self) -> None:
        service = ibb.IBBService(self.client)

        transport: ibb.service.IBBTransport
        protocol: asyncio.Protocol
        transport, protocol = await service.open_session(
            TestProtocol,
            self.args.j,
        )
        transport.write("Warren snores through the night like a bear—a bass to the treble of the loons.".encode('utf-8'))
        transport.close()
        e = await protocol.closed_fut
        if e is not None:
            print(f'error awaiting connection close: {e}', file=sys.stderr)
            sys.exit(1)
        # Echo the data we read back so that the other side can confirm that
        # what it sent is what was received.
        msg = Message(to=self.args.j, type_=MessageType.NORMAL)
        msg.body[None] = protocol.data.decode('utf-8')
        msg.doneibb = DoneIBB()
        await self.client.send(msg)


class RecvIBB(aioxmpp_client.Daemon):
    def __init__(self) -> None:
        super().__init__()

    def prepare_argparse(self) -> None:
        super().prepare_argparse()

        def jid(s):
            return JID.fromstr(s)
        self.argparse.add_argument(
            "-j",
            type=jid,
            help="The JID to expect an IBB session from",
        )
        self.argparse.add_argument(
            "-sid",
            type=str,
            help="The IBB session ID to expect",
        )

    async def run(self) -> None:
        # First send the other side of the connection our JID so that it knows
        # what JID to use.
        msg = Message(to=self.args.j, type_=MessageType.NORMAL)
        msg.startibb = StartIBB()
        await self.client.send(msg)

        service = ibb.IBBService(self.client)

        transport: ibb.service.IBBTransport
        protocol: asyncio.Protocol
        transport, protocol = await service.expect_session(
            TestProtocol,
            self.args.j,
            self.args.sid,
        )
        transport.write(b"I feel a deep security in the single-mindedness of freight trains.")
        e = await protocol.closed_fut
        if e is not None:
            print(f'error awaiting connection close: {e}', file=sys.stderr)
            sys.exit(1)
        # Echo the data we read back so that the other side can confirm that
        # what it sent is what was received.
        msg = Message(to=self.args.j, type_=MessageType.NORMAL)
        msg.body[None] = protocol.data.decode('utf-8')
        msg.doneibb = DoneIBB()
        e = await self.client.send(msg)


Message.startibb = xso.Child([StartIBB])
Message.doneibb = xso.Child([DoneIBB])
