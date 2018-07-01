// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package styling

import (
	"bufio"
	"bytes"
	"io"
)

// TODO: refactor to get rid of the state stuff and just use bools. I originally
// did the state byte because I needed to be able to rewind on occasion and
// didn't want to store all these values and all the previous values (it was
// easier to store just one byte and one previous state byte) but now that
// optimization is no longer necessary.
const (
	inPre = 1 << iota
	notLineStart
	nextInPre
	exitingPre
)

type preBlockParser struct {
	r     *bufio.Reader
	state uint8
}

func (p *preBlockParser) Style() Style {
	if p.state&inPre != 0 {
		return PreBlock
	}
	return 0
}

func (p *preBlockParser) Read(b []byte) (n int, err error) {
	if p.state&nextInPre != 0 {
		p.state = (p.state &^ nextInPre) | inPre
	}

	for i := 0; i < len(b); i++ {
		// If we're at the start of a line
		if p.state&notLineStart == 0 {
			// If we're not already in a pre block, look for "```" (start pre)
			if p.state&inPre == 0 {
				peek, err := p.r.Peek(3)
				switch err {
				case bufio.ErrBufferFull:
					return n, nil
				case nil, io.EOF:
				default:
					return n, err
				}

				// We found the start of a pre block.
				if bytes.Equal(peek, []byte("```")) {
					if i > 0 {
						p.state |= nextInPre
						return n, err
					}
					p.state = inPre
				}
			} else {
				// If we are in a pre block, look for "```\n" (end pre)
				peek, err := p.r.Peek(4)
				switch err {
				case bufio.ErrBufferFull:
					return n, nil
				case nil, io.EOF:
				default:
					return n, err
				}

				// We found the end of the pre block.
				if bytes.Equal(peek, []byte("```\n")) {
					p.state |= exitingPre
				}
			}
		}

		bb, err := p.r.ReadByte()
		if err != nil {
			return n, err
		}
		b[i] = bb
		n++
		if bb == '\n' {
			if p.state&exitingPre != 0 {
				p.state = p.state &^ (exitingPre | inPre)
			}
			p.state = p.state &^ notLineStart
		} else {
			p.state |= notLineStart
		}
	}

	return n, err
}
