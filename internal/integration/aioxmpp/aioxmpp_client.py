#!/usr/bin/python
# Copyright 2022 The Mellium Contributors.
# Use of this source code is governed by the BSD 2-clause
# license that can be found in the LICENSE file.

import abc
import argparse
import asyncio
import configparser
import sys

import aioxmpp

###
# This script acts as a wrapper around aioxmpp that starts a simple client.
# It is not meant to do any testing itself, instead use the Go aioxmpp.Import option to create a test script that
# subclasses Daemon and overrides its run method.
###


class Daemon(metaclass=abc.ABCMeta):
    def __init__(self) -> None:
        super().__init__()
        self.argparse: argparse.ArgumentParser = argparse.ArgumentParser()

    def prepare_argparse(self) -> None:
        self.argparse.add_argument(
            "-c",
            default="aioxmpp_config.ini",
            type=argparse.FileType("r"),
            help="The config file to load",
        )

    def configure(self):
        self.args: argparse.Namespace = self.argparse.parse_args()
        self.config: configparser.ConfigParser = configparser.ConfigParser()
        self.config.read_file(self.args.c)
        self.security: aioxmpp.security_layer.SecurityLayer = aioxmpp.make_security_layer(
            self.config.get("client", "password"),
            no_verify=True,
        )
        self.jid: aioxmpp.JID = aioxmpp.JID.fromstr(self.config.get("client", "jid"))
        xmpp_port = self.config.getint("client", "port")
        conn: aioxmpp.connector.BaseConnector = aioxmpp.connector.STARTTLSConnector()
        self.client: aioxmpp.Client = aioxmpp.PresenceManagedClient(
            self.jid,
            self.security,
            override_peer=[("127.0.0.1", xmpp_port, conn)],
        )

    async def run_test(self) -> None:
        async with self.client.connected():
            await self.run()

    async def run(self) -> None:
        raise NotImplementedError("test file must override run function")


async def run_test(cls):
    # We create the class in a wrapper function so that the aioxmpp.Client isn't created until after the event loop
    # created by the call to asyncio.run is created. Otherwise the client creates its own event loop if one isn't
    # already running and we end up trying to run things on the wrong loop.
    instance = cls()
    instance.prepare_argparse()
    instance.configure()
    await instance.run_test()

if __name__ == "__main__":
    print("running {{ len .Imports }} aioxmpp tests…")
{{- range $idx, $script := .Imports }}
    def runner():
        from {{ index $script 0 }} import {{ index $script 1 }}
        asyncio.run(run_test({{ index $script 1}}))
    runner()
{{- end }}
