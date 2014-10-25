# Koiné

**Koiné** is an XMPP JID address library and validator for Go which aims to be
fully [RFC 6122][rfc6122] compliant, except using [IDNA2008][idna2008] for
normalizing domain names.

To use it in your project, import it like so:

```go
import github.com/SamWhited/koine
```

## Status

A test suite is being developed and then the actual library will be brought up
to date. Basic functionality and unicode normalization is present, but the
library is currently not IDNA2008 compliant and is using a limited
implementation of stringprep.

## License

Copyright 2014 Sam Whited.
Use of this source code is governed by the BSD 2-clause license that can be
found in the LICENSE file.

[rfc6122]: https://www.rfc-editor.org/rfc/rfc6122.txt
[idna2008]: http://www.unicode.org/reports/tr46/#IDNA2008
