# Message Styling Tests

This package was written in part to act as a reference implementation for
[XEP-0393: Message Styling].
To this end, the tests are designed to be exportable to a language agnostic
format.
For more information, see the file `export_test.go`.
To export the tests run the following in this directory:

    go test -tags export

This will result in the creation of the file `decoder_tests.json`.
This file will be a JSON array of test cases.
Each case has the following properties:

- "Name": a string description of the test
- "Input": the message styling string to parse
- "Tokens": an array of tokens that result from parsing the string

Each token has the following properties:

- Mask: a numeric bitmask containing all of the styles applied to the token
- Data: the subslice of the input string that was detected as a token
- Info: any info string present at the start of pre-formatted blocks
- Quote: the numeric quotation depth of the token if inside a block quote

The values for "Mask" can be found in the "[constants]" section of the package
documentation.

## Known Limitations

The bitmask contains several styles, such as `BlockPreStart`, that are not part
of the specification and may not be used by all implementations.
These are to mark the start and end of spans and blocks and may be ignored if
your implementation does not differentiate them.

Long plain text spans may also be broken up at arbitrary intervals depending on
the parser buffer length.
For example, the string "one two" could be a single token "one two" or it could
be broken up into the tokens "one " and "two" or "one" and " two" or even "on"
and "e two" by the tests.
These test cases should easily fit within any reasonable buffer, but if your
implementation uses a smaller buffer size or breaks up long spans differently
you may have to account for this when running these test cases.

[XEP-0393: Message Styling]: https://xmpp.org/extensions/xep-0393.html
[constants]: https://pkg.go.dev/mellium.im/xmpp/styling#pkg-constants
