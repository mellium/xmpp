# XEPs

This library implements an number of XMPP Extension Protocols (XEPs) and RFCs.
Others may be implemented in third party libraries.

| RFC       | Package     |
| ----------| ----------- |
| [RFC5122] | [uri]       |
| [RFC6120] | [xmpp]¹     |
| [RFC6121] | [xmpp]¹     |
| [RFC7590] | [xmpp]¹     |
| [RFC7622] | [jid]       |

| XEP                                                         | Package     |
| ----------------------------------------------------------- | ----------- |
| [XEP-0066: Out of Band Data]                                | [oob]       |
| [XEP-0082: XMPP Date and Time Profiles]                     | [xtime]     |
| [XEP-0106: JID Escaping]                                    | [jid]       |
| [XEP-0114: Jabber Component Protocol]                       | [component] |
| [XEP-0138: Stream Compression]                              | [compress]  |
| [XEP-0156: Discovering Alternative XMPP Connection Methods] | [dial]      |
| [XEP-0184: Message Delivery Receipts]                       | [receipts]  |
| [XEP-0199: XMPP Ping]                                       | [ping]      |
| [XEP-0202: Entity Time]                                     | [xtime]     |
| [XEP-0229: Stream Compression with LZW]                     | [compress]  |
| [XEP-0288: Bidirectional Server-to-Server Connections]      | [stream]    |
| [XEP-0392: Consistent Color Generation]                     | [color]     |
| [XEP-0393: Message Styling]                                 | [styling]   |

---

1. Functionality may be spread over several packages.

[RFC5122]: https://tools.ietf.org/html/rfc5122
[RFC6120]: https://tools.ietf.org/html/rfc6120
[RFC6121]: https://tools.ietf.org/html/rfc6121
[RFC7590]: https://tools.ietf.org/html/rfc7590
[RFC7622]: https://tools.ietf.org/html/rfc7622

[XEP-0066: Out of Band Data]: https://xmpp.org/extensions/xep-0066.html
[XEP-0082: XMPP Date and Time Profiles]: https://xmpp.org/extensions/xep-0030.html
[XEP-0106: JID Escaping]: https://xmpp.org/extensions/xep-0106.html
[XEP-0114: Jabber Component Protocol]: https://xmpp.org/extensions/xep-0114.html
[XEP-0138: Stream Compression]: https://xmpp.org/extensions/xep-0138.html
[XEP-0156: Discovering Alternative XMPP Connection Methods]: https://xmpp.org/extensions/xep-0156
[XEP-0184: Message Delivery Receipts]: https://xmpp.org/extensions/xep-0184.html
[XEP-0199: XMPP Ping]: https://xmpp.org/extensions/xep-0199.html
[XEP-0202: Entity Time]: https://xmpp.org/extensions/xep-0202.html
[XEP-0229: Stream Compression with LZW]: https://xmpp.org/extensions/xep-0229.html
[XEP-0288: Bidirectional Server-to-Server Connections]: https://xmpp.org/extensions/xep-0288.html
[XEP-0392: Consistent Color Generation]: https://xmpp.org/extensions/xep-0392.html
[XEP-0393: Message Styling]: https://xmpp.org/extensions/xep-0393.html

[color]: https://pkg.go.dev/mellium.im/xmpp/color
[component]: https://pkg.go.dev/mellium.im/xmpp/component
[compress]: https://pkg.go.dev/mellium.im/xmpp/compress
[dial]: https://pkg.go.dev/mellium.im/xmpp/dial
[jid]: https://pkg.go.dev/mellium.im/xmpp/jid
[oob]: https://pkg.go.dev/mellium.im/xmpp/oob
[ping]: https://pkg.go.dev/mellium.im/xmpp/ping
[receipts]: https://pkg.go.dev/mellium.im/xmpp/receipts
[stream]: https://pkg.go.dev/mellium.im/xmpp/stream
[styling]: https://pkg.go.dev/mellium.im/xmpp/styling
[uri]: https://pkg.go.dev/mellium.im/xmpp/uri
[xmpp]: https://pkg.go.dev/mellium.im/xmpp/xmpp
[xtime]: https://pkg.go.dev/mellium.im/xmpp/xtime
