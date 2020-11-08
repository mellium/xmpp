// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package styling_test

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"mellium.im/xmpp/styling"
)

func TestCopyToken(t *testing.T) {
	tok := &styling.Token{
		Data: []byte("t"),
	}
	tok2 := tok.Copy()
	tok2.Data[0] = 'r'
	if tok.Data[0] == tok2.Data[0] {
		t.Errorf("Data was mutated, copy failed")
	}
}

type tokenAndStyle struct {
	styling.Token
	Quote uint
}

var decoderTestCases = []struct {
	name  string
	input string
	toks  []tokenAndStyle
	Err   error
}{
	{
		name: "plain blocks",
		input: `one
and two`,
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("one\n"),
				},
			},
			{
				Token: styling.Token{
					Data: []byte("and two"),
				},
			},
		},
	},
	{
		name:  "pre block with closing",
		input: "```\npre *fmt* ```\n```\nplain",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("```\n"),
					Mask: styling.BlockPre | styling.BlockPreStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("pre *fmt* ```\n"),
					Mask: styling.BlockPre,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("```\n"),
					Mask: styling.BlockPre | styling.BlockPreEnd,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("plain"),
				},
			},
		},
	},
	{
		name:  "pre block EOF",
		input: "```\na\n```",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("```\n"),
					Mask: styling.BlockPre | styling.BlockPreStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("a\n"),
					Mask: styling.BlockPre,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("```"),
					Mask: styling.BlockPre | styling.BlockPreEnd,
				},
			},
		},
	},
	{
		name:  "pre block no terminator EOF",
		input: "```\na```",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("```\n"),
					Mask: styling.BlockPre | styling.BlockPreStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("a```"),
					Mask: styling.BlockPre,
				},
			},
		},
	},
	{
		name:  "pre block no body EOF",
		input: "```newtoken\n",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("```newtoken\n"),
					Mask: styling.BlockPre | styling.BlockPreStart,
				},
			},
		},
	},
	{
		name: "single level block quote",
		input: `>  quoted
not quoted`,
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte(">  "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("quoted\n"),
					Mask: styling.BlockQuote,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Mask: styling.BlockQuote | styling.BlockQuoteEnd,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("not quoted"),
				},
			},
		},
	},
	{
		name: "multi level block quote",
		input: `>  quoted
>>   quote > 2
>quote 1

not quoted`,
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte(">  "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("quoted\n"),
					Mask: styling.BlockQuote,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte(">"),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte(">   "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 2,
			},
			{
				Token: styling.Token{
					Data: []byte("quote > 2\n"),
					Mask: styling.BlockQuote,
				},
				Quote: 2,
			},
			{
				Token: styling.Token{
					Mask: styling.BlockQuote | styling.BlockQuoteEnd,
				},
				Quote: 2,
			},
			{
				Token: styling.Token{
					Data: []byte(">"),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("quote 1\n"),
					Mask: styling.BlockQuote,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Mask: styling.BlockQuote | styling.BlockQuoteEnd,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("\n"),
				},
			},
			{
				Token: styling.Token{
					Data: []byte("not quoted"),
				},
			},
		},
	},
	{
		name:  "quote start then EOF",
		input: `> `,
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("> "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
		},
	},
	{
		name: "quote with children",
		input: "> ```" + `
> pre
> ` + "```" + `
> not pre`,
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("> "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("```\n"),
					Mask: styling.BlockQuote | styling.BlockPre | styling.BlockPreStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("> "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("pre\n"),
					Mask: styling.BlockQuote | styling.BlockPre,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("> "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("```\n"),
					Mask: styling.BlockQuote | styling.BlockPre | styling.BlockPreEnd,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("> "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("not pre"),
					Mask: styling.BlockQuote,
				},
				Quote: 1,
			},
		},
	},
	{
		name: "pre end of parent",
		input: "> ```" + `
> pre
plain`,
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("> "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("```\n"),
					Mask: styling.BlockQuote | styling.BlockPre | styling.BlockPreStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("> "),
					Mask: styling.BlockQuote | styling.BlockQuoteStart,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("pre\n"),
					Mask: styling.BlockQuote | styling.BlockPre,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Mask: styling.BlockQuote | styling.BlockQuoteEnd,
				},
				Quote: 1,
			},
			{
				Token: styling.Token{
					Data: []byte("plain"),
				},
			},
		},
	},
	{
		name:  "spans",
		input: "*strong* _emph_~strike~  `pre`",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("*"),
					Mask: styling.SpanStrong | styling.SpanStrongStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("strong"),
					Mask: styling.SpanStrong,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("*"),
					Mask: styling.SpanStrong | styling.SpanStrongEnd,
				},
			},
			{
				Token: styling.Token{
					Data: []byte(" "),
				},
			},
			{
				Token: styling.Token{
					Data: []byte("_"),
					Mask: styling.SpanEmph | styling.SpanEmphStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("emph"),
					Mask: styling.SpanEmph,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("_"),
					Mask: styling.SpanEmph | styling.SpanEmphEnd,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("~"),
					Mask: styling.SpanStrike | styling.SpanStrikeStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("strike"),
					Mask: styling.SpanStrike,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("~"),
					Mask: styling.SpanStrike | styling.SpanStrikeEnd,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("  "),
				},
			},
			{
				Token: styling.Token{
					Data: []byte("`"),
					Mask: styling.SpanPre | styling.SpanPreStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("pre"),
					Mask: styling.SpanPre,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("`"),
					Mask: styling.SpanPre | styling.SpanPreEnd,
				},
			},
		},
	},
	{
		name:  "spans lazily match",
		input: "*strong*plain*",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("*"),
					Mask: styling.SpanStrong | styling.SpanStrongStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("strong"),
					Mask: styling.SpanStrong,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("*"),
					Mask: styling.SpanStrong | styling.SpanStrongEnd,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("plain*"),
				},
			},
		},
	},
	{
		name:  "invalid diretives ignored",
		input: "* plain *strong*",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("* plain "),
				},
			},
			{
				Token: styling.Token{
					Data: []byte("*"),
					Mask: styling.SpanStrong | styling.SpanStrongStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("strong"),
					Mask: styling.SpanStrong,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("*"),
					Mask: styling.SpanStrong | styling.SpanStrongEnd,
				},
			},
		},
	},
	{
		name:  "end span only",
		input: "not strong*",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("not strong*"),
				},
			},
		},
	},
	{
		name:  "start span only",
		input: "*not strong",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("*not strong"),
				},
			},
		},
	},
	{
		name:  "span lines",
		input: "*not \n strong*",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("*not \n"),
				},
			},
			{
				Token: styling.Token{
					Data: []byte(" strong*"),
				},
			},
		},
	},
	{
		name:  "invalid end span",
		input: "*not *strong",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("*not *strong"),
				},
			},
		},
	},
	{
		name:  "empty span",
		input: "**",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("**"),
				},
			},
		},
	},
	{
		name:  "3 unmatched directives",
		input: "***",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("***"),
				},
			},
		},
	},
	{
		name:  "4 unmatched directives",
		input: "****",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("****"),
				},
			},
		},
	},
	{
		name:  "overlapping directives",
		input: "*this cannot _overlap*_",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("*"),
					Mask: styling.SpanStrong | styling.SpanStrongStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("this cannot _overlap"),
					Mask: styling.SpanStrong,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("*"),
					Mask: styling.SpanStrong | styling.SpanStrongEnd,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("_"),
				},
			},
		},
	},
	{
		name:  "pre cannot have children",
		input: "_no pre `with *children*`_",
		toks: []tokenAndStyle{
			{
				Token: styling.Token{
					Data: []byte("_"),
					Mask: styling.SpanEmph | styling.SpanEmphStart,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("no pre "),
					Mask: styling.SpanEmph,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("`"),
					Mask: styling.SpanPre | styling.SpanPreStart | styling.SpanEmph,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("with *children*"),
					Mask: styling.SpanPre | styling.SpanEmph,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("`"),
					Mask: styling.SpanPre | styling.SpanPreEnd | styling.SpanEmph,
				},
			},
			{
				Token: styling.Token{
					Data: []byte("_"),
					Mask: styling.SpanEmph | styling.SpanEmphEnd,
				},
			},
		},
	},
}

func TestToken(t *testing.T) {
	for _, tc := range decoderTestCases {
		// Make a copy of the expected tokens that we can pop from.
		toks := make([]tokenAndStyle, len(tc.toks))
		copy(toks, tc.toks)

		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.input)
			d := styling.NewDecoder(r)
			var n int
			for {
				tok, err := d.Token()
				switch err {
				case nil:
				case io.EOF:
					if tc.Err != nil {
						t.Errorf("Expected error but got io.EOF: %v", err)
					}
					if len(toks) != 0 {
						var tokStrs []string
						for _, tok := range toks {
							tokStrs = append(tokStrs, string(tok.Data))
						}
						t.Fatalf("Reached EOF at token %d, but expected remaining tokens: %+v (%v)", n, tc.toks, tokStrs)
					}
					return
				default:
					if tc.Err != err {
						t.Errorf("Unexpected error: want=%v, got=%v", tc.Err, err)
					}
				}

				if len(toks) == 0 {
					t.Fatalf("Did not expect more tokens but got %+v (%q), %v", tok, tok.Data, err)
				}

				var expectedTok tokenAndStyle
				expectedTok, toks = toks[0], toks[1:]
				if expectedTok.Mask != tok.Mask {
					t.Errorf("Unexpected mask for token %d: want=%#b, got=%#b", n, expectedTok.Mask, tok.Mask)
				}
				if !bytes.Equal(expectedTok.Data, tok.Data) {
					t.Errorf("Unexpected data for token %d: want=%q, got=%q", n, expectedTok.Data, tok.Data)
				}
				if d.Style() != expectedTok.Mask {
					t.Errorf("Unexpected decoder style after token %d: want=%#b, got=%#b", n, expectedTok.Mask, d.Style())
				}
				if d.Quote() != expectedTok.Quote {
					t.Errorf("Unexpected block quote level after token %d: want=%d, got=%d", n, expectedTok.Quote, d.Quote())
				}
				n++
			}
		})
	}
}

func TestScan(t *testing.T) {
	for _, tc := range decoderTestCases {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.input)
			s := bufio.NewScanner(r)
			s.Split(styling.Scan())
			var n int
			for s.Scan() {
				// Skip block quote end tokens in the test which aren't returned by the
				// scanner.
				if len(tc.toks[n].Data) == 0 {
					n++
				}
				if len(tc.toks) < n+1 {
					t.Fatalf("Expected to be done scanning, but Scan still returning: %q, %v", s.Bytes(), s.Err())
				}
				if data := tc.toks[n].Data; !bytes.Equal(data, s.Bytes()) {
					t.Errorf("Unexpected token while scanning: want=%q, got=%q", data, s.Bytes())
				}
				n++
			}
			if n != len(tc.toks) {
				t.Fatalf("Did not scan enough tokens, %d remain", len(tc.toks)-n)
			}
		})
	}
}

const eofReadData = "```test"

type EOFRead struct{}

func (EOFRead) Read(b []byte) (int, error) {
	data := []byte(eofReadData)
	n := copy(b, data)
	// We don't care if the copy doesn't actually complete, this is just to test
	// early EOF.
	return n, io.EOF
}

func TestEOFPre(t *testing.T) {
	d := styling.NewDecoder(EOFRead{})
	tok, err := d.Token()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal([]byte(eofReadData), tok.Data) {
		t.Errorf("Unexpected data: want=%q, got=%q", eofReadData, tok.Data)
	}
}

var testingErr = errors.New("an error")

type ErrRead struct{}

func (ErrRead) Read([]byte) (int, error) {
	return 0, testingErr
}

func TestDecodeErr(t *testing.T) {
	d := styling.NewDecoder(ErrRead{})
	_, err := d.Token()
	if err != testingErr {
		t.Errorf("Want error passed through, got %v", err)
	}
}

var blockSkipTestCases = [...]struct {
	input string
	pop   int
	token string
	err   error
	span  bool
}{
	0:  {input: "*one* two", err: io.EOF},
	1:  {input: "one *two*\nthree", token: "three"},
	2:  {input: "one *two*\nthree", pop: 1, token: "three"},
	3:  {input: "```test\none\ntwo", err: io.EOF},
	4:  {input: "```test\none\ntwo", pop: 1, err: io.EOF},
	5:  {input: "```test\none\ntwo\n```\nfour", err: io.EOF},
	6:  {input: "```test\none\ntwo\n```\nfour", pop: 1, token: "four"},
	7:  {input: "> test", err: io.EOF},
	8:  {input: "> one\ntwo", err: io.EOF},
	9:  {input: "> one\n>> two\n>three\nfour", pop: 1, token: "four"},
	10: {input: "> one\n>> two\n>three\nfour", pop: 3, token: "four"},
	11: {input: "> one\n>> two\n>three\nfour", pop: 4, token: ">"},
	12: {input: "> one\n>> two\n>three\nfour", pop: 6, token: "four"},
	13: {input: "> ```start\n>one\n>```\n>two\nthree", err: io.EOF},
	14: {input: "> ```start\n>one\n>```\n>two\nthree", pop: 1, token: "three"},

	15: {span: true, input: "*one* two", err: io.EOF},
	16: {span: true, input: "*one* two", pop: 1, token: " two"},
	17: {span: true, input: "*one _two_* three", pop: 1, token: " three"},
	18: {span: true, input: "*one _two_* three", pop: 3, token: "*"},
}

func TestSkip(t *testing.T) {
	for i, tc := range blockSkipTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := strings.NewReader(tc.input)
			d := styling.NewDecoder(r)
			for i := 0; i < tc.pop; i++ {
				_, err := d.Token()
				if err != nil {
					t.Fatalf("error at token %d: %v", i, err)
				}
			}
			var err error
			if tc.span {
				err = d.SkipSpan()
			} else {
				err = d.SkipBlock()
			}
			if err != tc.err {
				t.Errorf("final error does not match: want=%v, got=%v", tc.err, err)
			}
			if err != nil {
				return
			}
			tok, err := d.Token()
			if err != nil {
				t.Fatalf("error on expected token: %v", err)
			}
			if string(tok.Data) != tc.token {
				t.Errorf("Final token does not match: want=%q, got=%q", tc.token, tok.Data)
			}
		})
	}
}
