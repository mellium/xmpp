#!/usr/bin/python
# Copyright 2022 The Mellium Contributors.
# Use of this source code is governed by the BSD 2-clause
# license that can be found in the LICENSE file.

import abc
import argparse
import asyncio
import configparser
import sys

###
# This script acts as a wrapper around a testing script.
# It is not meant to do any testing itself, instead use the Go python.Import
# option to create a test script that subclasses Daemon and overrides its
# configure and run methods.
###


class Daemon(metaclass=abc.ABCMeta):
    """
    The Daemon class is an abstract base class used by all integration tests
    against Python libraries. It is meant to be subclassed by individual
    packages such as the aioxmpp package
    """
    def __init__(self) -> None:
        super().__init__()
        self.argparse: argparse.ArgumentParser = argparse.ArgumentParser()
        self.config: configparser.ConfigParser = configparser.ConfigParser()

    def prepare_argparse(self) -> None:
        """
        The prepare_argparse method can be overridden in a subclass to add
        custom arguments to the Python script used by tests.
        """
        self.argparse.add_argument(
            "-c",
            default="python_config.ini",
            type=argparse.FileType("r"),
            help="The config file to load",
        )

    def configure(self):
        """
        The configure method parses arguments and reads the config file.
        It should be overridden by a subclass to also construct the XMPP client
        or other resources needed by the test.
        """
        self.args: argparse.Namespace = self.argparse.parse_args()
        self.config.read_file(self.args.c)

    async def run_test(self) -> None:
        """
        The run_test method should be overridden by the test script or another
        testing helper package.
        """
        raise NotImplementedError("test file must override run function")


async def run_test(cls):
    instance = cls()
    instance.prepare_argparse()
    instance.configure()
    await instance.run_test()

if __name__ == "__main__":
    print("running {{ len .Imports }} python scripts…", file=sys.stderr)
{{- range $idx, $script := .Imports }}
    def runner():
        from {{ index $script 0 }} import {{ index $script 1 }}
        asyncio.run(run_test({{ index $script 1}}))
    runner()
{{- end }}
