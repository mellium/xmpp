// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ibb

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"net"
	"time"

	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

type stanzaReadWriter struct {
	s      *xmpp.Session
	sid    string
	stanza string
	seq    uint16
	to     jid.JID
}

func newStanzaReadWriter(to jid.JID, s *xmpp.Session, sid string, stanza string) *stanzaReadWriter {
	return &stanzaReadWriter{
		s:      s,
		sid:    sid,
		stanza: stanza,
		to:     to,
	}
}

func (w *stanzaReadWriter) Write(p []byte) (int, error) {
	data := dataPayload{
		Seq:  w.seq,
		SID:  w.sid,
		data: p,
	}

	var err error
	switch w.stanza {
	case messageType:
		err = w.s.Encode(dataMessage{
			Message: stanza.Message{
				To: w.to,
			},
			Data: data,
		})
	case iqType:
		err = w.s.Encode(dataIQ{
			IQ: stanza.IQ{
				To: w.to,
			},
			Data: data,
		})
	}
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Conn is an IBB stream.
// Writes to the stream are buffered up to blocksize and calling Close forces
// any remaining data to be flushed.
type Conn struct {
	closed        bool
	readBuf       *bufio.Reader
	readDeadline  time.Time
	remoteAddr    jid.JID
	seq           uint16
	sid           string
	stanza        string
	s             *xmpp.Session
	writeBuf      *bufio.Writer
	writeDeadline time.Time
	closeFunc     func() error
	handler       *Handler
}

func newConn(h *Handler, s *xmpp.Session, iq openIQ) *Conn {
	rwc := newStanzaReadWriter(iq.IQ.To, s, iq.Open.SID, iq.Open.Stanza)
	b64Reader := base64.NewDecoder(base64.StdEncoding, rwc)
	b64Writer := base64.NewEncoder(base64.StdEncoding, rwc)
	r := bufio.NewReaderSize(b64Reader, int(iq.Open.BlockSize))
	w := bufio.NewWriterSize(b64Writer, int(iq.Open.BlockSize))

	return &Conn{
		readBuf:    r,
		remoteAddr: iq.IQ.To,
		sid:        iq.Open.SID,
		s:          s,
		stanza:     iq.Open.Stanza,
		writeBuf:   w,
		closeFunc:  b64Writer.Close,
		handler:    h,
	}
}

// SID returns a unique session ID for the connection.
func (c *Conn) SID() string {
	return c.sid
}

// Stanza returns the carrier stanza type ("message" or "iq") for payloads
// received by the IBB session.
func (c *Conn) Stanza() string {
	return c.stanza
}

// Read reads data from the IBB stream.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	if c.closed {
		return 0, io.EOF
	}
	return c.readBuf.Read(b)
}

// Write writes data to the IBB stream.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *Conn) Write(b []byte) (n int, err error) {
	if c.closed {
		return 0, io.EOF
	}
	return c.writeBuf.Write(b)
}

// LocalAddr returns the local network address of the underlying XMPP session.
func (c *Conn) LocalAddr() net.Addr {
	return c.s.LocalAddr()
}

// RemoteAddr returns the remote network address of the IBB stream.
func (c *Conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// Size returns the blocksize for the underlying buffer when writing to the IBB
// stream.
func (c *Conn) Size() int {
	return c.writeBuf.Size()
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
// If the write buffer contains data it will be written regardless of the
// blocksize.
func (c *Conn) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true

	c.handler.rmStream(c.sid)

	// Flush any remaining data to be written.
	err := c.writeBuf.Flush()
	if err != nil {
		return err
	}

	err = c.closeFunc()
	if err != nil {
		return err
	}

	// TODO: should we always at least try to send the close IQ even if something
	// else returns an error?
	respReadCloser, err := c.s.SendIQElement(context.TODO(), closePayload(c.sid), stanza.IQ{
		To:   c.remoteAddr,
		Type: stanza.SetIQ,
	})
	// TODO: how do we handle this error? Do we care if it errors?
	respReadCloser.Close()
	return err
}

// SetDeadline sets the read and write deadlines associated with the connection.
// It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *Conn) SetDeadline(t time.Time) error {
	c.readDeadline = t
	c.writeDeadline = t
	return nil
}

// SetReadDeadline sets the deadline for future Read calls and any
// currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that some of the
// data was successfully written.
// A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = t
	return nil
}
