# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

### Breaking

- stanza: change API of `WrapIQ` and `WrapPresence` to not abuse pointers
- xmpp: add new `SendIQ` API and remove response from `Send` and `SendElement`
- xmpp: use underlying net.Dialer's DNS Resolver in Dialer.
