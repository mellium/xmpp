// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package styling implements XEP-0393: Message Styling, a Markdown-like styling
// language.
package styling

import (
	"bufio"
	"bytes"
	"io"
	"unicode"
	"unicode/utf8"
)

// Style is a bitmask that represents a set of styles that can be applied to
// text.
type Style uint32

// The style bits.
const (
	// BlockPre represents a preformatted text block.
	// It should be displayed in a monospace font with no change to line breaks.
	BlockPre Style = 1 << iota

	// BlockQuote represents a nestable quotation.
	// To get the level of the quotation see the Quote method.
	BlockQuote

	// SpanEmph is an inline span of text that should be displayed in italics.
	SpanEmph

	// SpanStrong is an inline span of text that should be displayed bold.
	SpanStrong

	// SpanStrike is an inline span of text that should be displayed with a
	// horizontal line through the middle (strike through).
	SpanStrike

	// SpanPre is an inline span of text that should be displayed in a monospace
	// font.
	SpanPre

	// Styling directive markers.
	// It is often desirable to distinguish the characters that triggered styling
	// from surrounding text. These bits are set only on styling directives, the
	// characters or sequences of characters that result in the style changing.
	// The corresponding style bit will also be set whenever the start or end bits
	// are set. For example, in *strong* the first "*" will have
	// SpanStrong|SpanStrongStart set, the "strong" will only have SpanStrong set,
	// and the last "*" will have SpanStrong|SpanStrongEnd set.
	BlockPreStart
	BlockPreEnd
	BlockQuoteStart
	BlockQuoteEnd
	SpanEmphStart
	SpanEmphEnd
	SpanStrongStart
	SpanStrongEnd
	SpanStrikeStart
	SpanStrikeEnd
	SpanPreStart
	SpanPreEnd

	// Various useful masks
	// These bitmasks are provided as a convenience to make it easy to check what
	// general category of styles are applied.
	Block               = BlockPre | BlockQuote
	Span                = SpanEmph | SpanStrong | SpanStrike | SpanPre
	SpanEmphDirective   = SpanEmphStart | SpanEmphEnd
	SpanStrongDirective = SpanStrongStart | SpanStrongEnd
	SpanStrikeDirective = SpanStrikeStart | SpanStrikeEnd
	SpanPreDirective    = SpanPreStart | SpanPreEnd
	SpanDirective       = SpanEmphDirective | SpanStrongDirective | SpanStrikeDirective | SpanPreDirective
	SpanStartDirective  = SpanEmphStart | SpanStrongStart | SpanStrikeStart | SpanPreStart
	SpanEndDirective    = SpanEmphEnd | SpanStrongEnd | SpanStrikeEnd | SpanPreEnd
	BlockPreDirective   = BlockPreStart | BlockPreEnd
	BlockQuoteDirective = BlockQuoteStart | BlockQuoteEnd
	BlockDirective      = BlockPreDirective | BlockQuoteDirective
	BlockStartDirective = BlockPreStart | BlockQuoteStart
	BlockEndDirective   = BlockPreEnd | BlockQuoteEnd
	Directive           = SpanDirective | BlockDirective
	StartDirective      = SpanStartDirective | BlockStartDirective
	EndDirective        = SpanEndDirective | BlockEndDirective
)

// A Token represents a styling directive or unstyled span of text.
type Token struct {
	Mask Style
	Data []byte
}

// Copy creates a new copy of the token.
func (t Token) Copy() Token {
	data := make([]byte, len(t.Data))
	copy(data, t.Data)
	t.Data = data
	return t
}

var fence = []byte{'`', '`', '`'}

func isSpace(r rune) bool {
	return unicode.IsSpace(r) || unicode.Is(unicode.Space, r)
}

// Scan returns a new stateful split function for a bufio.Scanner that splits on
// message styling tokens.
// This function is different from a Decoder in that it will not return block
// quote end tokens.
//
// This is a low-level building block of this package.
// Most users will want to use a Decoder instead.
func Scan() bufio.SplitFunc {
	return (&Decoder{}).scan
}

// A Decoder represents a styling lexer reading a particular input stream.
// The parser assumes that input is encoded in UTF-8.
type Decoder struct {
	s             *bufio.Scanner
	clearMask     Style
	mask          Style
	quoteSplit    *Decoder
	quoteStarted  bool
	lastNewline   bool
	hasRun        bool
	spanStack     []byte
	skipByte      bool
	bufferedToken *Token
}

// NewDecoder creates a new styling parser reading from r.
// If r does not implement io.ByteReader, NewDecoder will do its own buffering.
func NewDecoder(r io.Reader) *Decoder {
	d := &Decoder{
		s: bufio.NewScanner(r),
	}
	d.s.Split(d.scan)
	return d
}

// Token returns the next styling token in the input stream.
// At the end of the input stream, token returns nil, io.EOF.
// Returned tokens do not always correspond directly to the input stream.
// For example, at the end of a block quote an empty token is returned with the
// mask BlockQuote|BlockQuoteEnd, but there is no explicit block quote
// terminator character in the input stream so its data will be empty.
//
// Slices of bytes in the returned token data refer to the parser's internal
// buffer and remain valid only until the next call to Token.
// To acquire a copy of the bytes call the token's Copy method.
func (d *Decoder) Token() (Token, error) {
	if d.bufferedToken != nil {
		t := *d.bufferedToken
		d.bufferedToken = nil
		return t, nil
	}

	prevLevel := d.Quote()
	if s := d.s.Scan(); !s {
		if err := d.s.Err(); err != nil {
			return Token{}, err
		}
		return Token{}, io.EOF
	}

	t := Token{
		Mask: d.Style(),
		Data: d.s.Bytes(),
	}

	// If we've dropped a block quote level, insert a token to indicate that we're
	// at the end of the quote since block quotes have no explicit terminator.
	currLevel := d.Quote()
	if currLevel < prevLevel {
		d.bufferedToken = &t
		return Token{Mask: d.Style()}, nil
	}

	return t, nil
}

// SkipSpan pops tokens from the decoder until it reaches the end of the current
// span, positioning the token stream at the beginning of the next span or
// block.
// If SkipSpan is called while no span is entered it behaves like SkipBlock.
// It returns any errors it encounters along the way.
func (d *Decoder) SkipSpan() error {
	for {
		prevLevel := d.Quote()
		tok, err := d.Token()
		if err != nil {
			return err
		}

		switch {
		case tok.Mask&StartDirective&^BlockQuoteStart > 0 || (tok.Mask&BlockQuoteStart == BlockQuoteStart && d.Quote() > prevLevel):
			if err := d.SkipSpan(); err != nil {
				return err
			}
		case tok.Mask&EndDirective > 0 || (tok.Mask == 0 && d.lastNewline):
			return nil
		}
	}
}

// SkipBlock pops tokens from the decoder until it reaches the end of the
// current block, positioning the token stream at the beginning of the next
// block.
// It returns any errors it encounters along the way.
// If SkipBlock is called at the beginning of the input stream before any tokens
// have been popped or any blocks have been entered, it will skip the entire
// input stream as if everything were contained in an imaginary "root" block.
func (d *Decoder) SkipBlock() error {
	for {
		prevLevel := d.Quote()
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch {
		case tok.Mask&BlockStartDirective&^BlockQuoteStart > 0 || (tok.Mask&BlockQuoteStart == BlockQuoteStart && d.Quote() > prevLevel):
			// If we're a start directive (other than starting a block quote), or
			// we're a block quote start directive that's deeper than the previous
			// level of block quote (ie. starting a new blockquote, not continuing an
			// existing one), recurse down into the inner block, skipping tokens.
			if err := d.SkipBlock(); err != nil {
				return err
			}
		case tok.Mask&BlockEndDirective > 0 || (tok.Mask == 0 && d.lastNewline):
			// If this is an end directive (or the end of a plain block), we're done
			// with skipping the current level, so end the current level of recursion.
			return nil
		}
	}
}

func (d *Decoder) scan(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	defer func() {
		if len(token) > 0 && token[len(token)-1] == '\n' {
			d.lastNewline = true
		}
	}()
	if d.lastNewline {
		tmpSplit := d
		for {
			tmpSplit.quoteStarted = false
			tmpSplit.hasRun = false
			if tmpSplit.quoteSplit == nil {
				break
			}
			tmpSplit = tmpSplit.quoteSplit
		}
		d.mask &^= BlockQuote
		d.lastNewline = false
	}
	d.mask &^= d.clearMask
	d.clearMask = 0

	switch {
	case len(d.spanStack) > 0:
		// If we're already in a span, just keep parsing it.
		return d.scanSpan(data, atEOF)
	case d.mask&BlockPre == BlockPre:
		// If we're inside a preblock, ignore everything else and scan lines until
		// the end of the preblock.
		d.hasRun = true
		return d.scanPre(data, atEOF)
	}

	// Look for new blocks
	switch l := startsBlockQuote(data); {
	case l > 0 && !d.quoteStarted:
		// If we haven't yet consumed our block quote start token, do so.
		d.mask |= BlockQuote | BlockQuoteStart
		d.clearMask |= BlockQuoteStart
		// Setup the inner parser if we haven't done so already.
		if d.quoteSplit == nil {
			d.quoteSplit = &Decoder{}
		}
		d.quoteStarted = true
		d.hasRun = true
		return l, data[:l], nil
	case l > 0 && d.quoteStarted:
		// If we've already consumed a quote start token and we encounter another,
		// let the inner Decoder handle it.
		return d.quoteSplit.scan(data, atEOF)
	case l == 0 && d.quoteSplit != nil:
		// If we're already in a block quote, delegate to the inner Decoder.
		if d.quoteStarted {
			return d.quoteSplit.scan(data, atEOF)
		}
		// If we're not in a block and we couldn't find a block start at our
		// level, we've dropped a level so reset the inner Decoder.
		d.quoteSplit = nil
	}
	d.hasRun = true

	newLineIDX := bytes.IndexByte(data, '\n')
	if bytes.HasPrefix(data, fence) {
		// A fenced code block is starting:
		d.mask |= BlockPre | BlockPreStart
		d.clearMask |= BlockPreStart
		switch {
		case newLineIDX > 0:
			// The full line is the codeblock start or end element or an invalid end
			// element that is actually a plain block that just starts with "```"
			return newLineIDX + 1, data[:newLineIDX+1], nil
		case atEOF:
			// We're at the end of the file, so the token is just the remainder of the
			// data.
			return len(data), data, nil
		}
	}

	return d.scanSpan(data, atEOF)
}

// Quote is the blockquote depth at the current position in the document.
func (d *Decoder) Quote() uint {
	var level uint
	if d.quoteStarted {
		level = 1 + d.quoteSplit.Quote()
	}
	// If the next token is a virtual block quote end token, pretend we haven't
	// dropped down a level yet.
	if d.bufferedToken != nil {
		level++
	}
	return level
}

// Style returns a bitmask representing the currently applied styles at the
// current position in the document.
func (d *Decoder) Style() Style {
	if d.bufferedToken != nil {
		return BlockQuoteEnd | BlockQuote
	}
	if d.hasRun {
		if d.quoteSplit != nil {
			return d.mask | d.quoteSplit.Style()
		}
		return d.mask
	}
	return 0
}

// scanPre is a split function that's used inside pre blocks where we don't have
// to bother parsing any children.
// All it does is look for the end of the pre block and line breaks.
func (d *Decoder) scanPre(data []byte, atEOF bool) (advance int, token []byte, err error) {
	switch idx := bytes.Index(data, fence); {
	case idx == 0 && !atEOF && len(data) == len(fence):
		// We need to make sure it's followed by a newline, so get more data.
		return 0, nil, nil
	case idx == 0 && (atEOF || (len(data) > len(fence) && data[len(fence)] == '\n')):
		d.mask |= BlockPreEnd
		d.clearMask |= BlockPre | BlockPreEnd
		l := len(fence)
		if !atEOF {
			l++
		}
		return l, data[:l], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	newLineIDX := bytes.IndexByte(data, '\n')
	if newLineIDX >= 0 {
		return newLineIDX + 1, data[:newLineIDX+1], nil
	}
	return 0, nil, nil
}

// scanSpan is a split function that finds and splits valid formatted spans.
func (d *Decoder) scanSpan(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Look for span styling directives
	startIDX := -1
	var startDirective byte

	for i, b := range data {
		// If we hit the end of the block without encountering any styling
		// directives, the whole span is a plain text span so just return it.
		if b == '\n' {
			return i + 1, data[:i+1], nil
		}

		if b != '*' && b != '_' && b != '`' && b != '~' {
			// This is not a styling directive. Nothing to do here.
			continue
		}

		nextRune, _ := utf8.DecodeRune(data[i+1:])
		nextSpace := isSpace(nextRune)
		prevRune, _ := utf8.DecodeLastRune(data[:i])
		prevSpace := isSpace(prevRune)

		switch {
		case d.skipByte:
			d.skipByte = false
			continue
		case len(d.spanStack) > 0 && b == d.spanStack[len(d.spanStack)-1]:
			// If this is an end directive that matches an outer spans start
			// directive, return the directive, or the rest of the span and clear the
			// mask.
			if i == 0 {
				switch b {
				case '*':
					d.mask |= SpanStrongEnd
					d.clearMask |= SpanStrong | SpanStrongEnd
				case '_':
					d.mask |= SpanEmphEnd
					d.clearMask |= SpanEmph | SpanEmphEnd
				case '~':
					d.mask |= SpanStrikeEnd
					d.clearMask |= SpanStrike | SpanStrikeEnd
				case '`':
					d.mask |= SpanPreEnd
					d.clearMask |= SpanPre | SpanPreEnd
				}
				d.spanStack = d.spanStack[:len(d.spanStack)-1]
				return 1, data[0:1], nil
			}

			// Return whatever remains inside the span.
			return i, data[:i], nil
		case startIDX == -1 && (d.mask&SpanPre == 0):
			// If we haven't found a start directive yet, this directive might be one
			// (unless we're already in a pre-span which doesn't allow children) so
			// check if it matches all the rules defined in XEP-0393 §6.2.
			// if we're at the start of the data (start of the line or just after a
			// previous styling directive) or we're immediately after a space, this
			// is a valid start styling directive.
			if (i == 0 || prevSpace) && !nextSpace {
				startIDX = i
				startDirective = b
				// Special case for a directive between two other matching directives
				// (ie. "***"). There is almost certainly a better way to handle this
				// that doesn't require special casing it.
				if len(data) > i+2 && data[i+1] == b && data[i+2] == b {
					// Skip one byte during this loop, then skip it again during the next
					// loop so that we find the real end token.
					// This is dumb and needs to be fixed.
					d.skipByte = true
					defer func() {
						d.skipByte = true
					}()
				}
				continue
			}
		case b == startDirective && !prevSpace && i > startIDX+1:
			// If we have already found a start directive, scan for a matching end
			// directive.

			// If we find one this is a valid span so return the previous plain span
			// (if any):
			if startIDX > 0 {
				return startIDX, data[:startIDX], nil
			}

			// Then add it to the stack:
			d.spanStack = append(d.spanStack, b)

			// or return the styling directive itself and set the mask.
			switch b {
			case '*':
				d.mask |= SpanStrong | SpanStrongStart
				d.clearMask |= SpanStrongStart
			case '_':
				d.mask |= SpanEmph | SpanEmphStart
				d.clearMask |= SpanEmphStart
			case '~':
				d.mask |= SpanStrike | SpanStrikeStart
				d.clearMask |= SpanStrikeStart
			case '`':
				d.mask |= SpanPre | SpanPreStart
				d.clearMask |= SpanPreStart
			}
			d.skipByte = false
			return 1, data[0:1], nil
		}
	}

	// If we didn't find a span, request new data until we find a span or newline.
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// startsBlockQuote looks for a block quote start token at the beginning of data
// and returns its length (or 0 if one was not found).
func startsBlockQuote(data []byte) int {
	if len(data) == 0 || data[0] != '>' {
		return 0
	}

	data = data[1:]
	l := 1
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if !isSpace(r) {
			return l
		}
		l += size
		data = data[size:]
	}
	return l
}
