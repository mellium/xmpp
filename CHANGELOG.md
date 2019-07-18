# Changelog

All notable changes to this project will be documented in this file.

## v0.12.0

### Breaking

- dial: moved network dialing types and functions into new package.
- dial: use underlying net.Dialer's DNS Resolver in Dialer.
- stanza: change API of `WrapIQ` and `WrapPresence` to not abuse pointers
- xmpp: add new `SendIQ` API and remove response from `Send` and `SendElement`

### Fixed

- xmpp: let `Session.Close` operate concurrently with `SendElement` et al.
