# Changelog

All notable changes to this project will be documented in this file.


## Unreleased

### Changed

- stanza: a zero value `IQType` now marshals as "get".


## v0.13.0 — 2019-07-27

### Breaking

- xmpp: change [`Handler`] to take an [`xmlstream.TokenReadEncoder`]
- xmpp: replace `EncodeToken` and `Flush` with [`TokenWriter`]
- xmpp: replace `Token` with [`TokenReader`]


### Added

- examples/echobot: add graceful shutdown on SIGINT
- xmpp: [`Encode`] and [`EncodeElement`] methods


### Changed

- xmpp: calls to `Serve` no longer return [`io.EOF`] on success


### Fixed

- examples/echobot: calling `Send` from within the handler resulted in deadlock
- xmpp: closing the input stream was racy, resulting in invalid XML


[`EncodeElement`]: https://godoc.org/mellium.im/xmpp#Session.EncodeElement
[`Encode`]: https://godoc.org/mellium.im/xmpp#Session.Encode
[`Handler`]: https://godoc.org/mellium.im/xmpp#Handler
[`io.EOF`]: https://golang.org/pkg/io/#EOF
[`TokenReader`]: https://godoc.org/mellium.im/xmpp#Session.TokenReader
[`TokenWriter`]: https://godoc.org/mellium.im/xmpp#Session.TokenWriter
[`xmlstream.TokenReadEncoder`]: https://godoc.org/mellium.im/xmlstream#TokenReadEncoder


## v0.12.0

### Breaking

- dial: moved network dialing types and functions into new package.
- dial: use underlying net.Dialer's DNS Resolver in Dialer.
- stanza: change API of `WrapIQ` and `WrapPresence` to not abuse pointers
- xmpp: add new `SendIQ` API and remove response from `Send` and `SendElement`
- xmpp: new API for writing custom tokens to a session

### Fixed

- xmpp: let `Session.Close` operate concurrently with `SendElement` et al.
