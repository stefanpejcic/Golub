// SPDX-License-Identifier: GPL-3.0-or-later

// Package mbox reads and writes mboxrd files and also accepts a single .eml.
package mbox

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
)

// MaxMessage caps one message so a broken file cannot pin memory.
const MaxMessage = 50 << 20

// ErrTooLarge is returned for a message over MaxMessage.
var ErrTooLarge = errors.New("mbox: message too large")

// A separator is "From sender date"; requiring the date keeps unescaped body lines that start with "From " intact.
var separatorRe = regexp.MustCompile(`^From \S+ .*\d{4}( [+-]\d{4})?\s*$`)

func isSeparator(line []byte) bool {
	return bytes.HasPrefix(line, []byte("From ")) && separatorRe.Match(line)
}

// Read calls fn with each message, lines ending in CRLF. A file that does not start with a
// "From " line is taken as one message.
func Read(r io.Reader, fn func(msg []byte) error) error {
	br := bufio.NewReaderSize(r, 64<<10)
	var cur bytes.Buffer
	started, isMbox, prevBlank := false, false, true
	flush := func() error {
		if !started {
			return nil
		}
		msg := cur.Bytes()
		// The blank line before the next separator belongs to the format, not the message.
		if isMbox {
			msg = bytes.TrimSuffix(msg, []byte("\r\n\r\n"))
			if !bytes.HasSuffix(msg, []byte("\r\n")) {
				msg = append(msg, '\r', '\n')
			}
		}
		err := fn(bytes.Clone(msg))
		cur.Reset()
		return err
	}
	first := true
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			body := bytes.TrimRight(line, "\r\n")
			if first {
				isMbox = isSeparator(body)
				first = false
			}
			switch {
			case isMbox && prevBlank && isSeparator(body):
				if ferr := flush(); ferr != nil {
					return ferr
				}
				started = true
			default:
				started = true
				if isMbox && bytes.HasPrefix(bytes.TrimLeft(body, ">"), []byte("From ")) && body[0] == '>' {
					body = body[1:]
				}
				if cur.Len()+len(body) > MaxMessage {
					return ErrTooLarge
				}
				cur.Write(body)
				if line[len(line)-1] == '\n' || err == io.EOF {
					cur.WriteString("\r\n")
				}
			}
			prevBlank = len(body) == 0
		}
		if errors.Is(err, io.EOF) {
			return flush()
		}
		if err != nil {
			return err
		}
	}
}

// Writer writes mboxrd with LF line endings, as mail tools expect.
type Writer struct {
	w io.Writer
}

// NewWriter starts an mbox on w.
func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

// Write appends one message.
func (mw *Writer) Write(msg []byte, from string, date time.Time) error {
	if from == "" || strings.ContainsAny(from, " \r\n") {
		from = "MAILER-DAEMON"
	}
	var b bytes.Buffer
	b.WriteString("From " + from + " " + date.UTC().Format(time.ANSIC) + "\n")
	text := strings.ReplaceAll(string(msg), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, ">"), "From ") {
			b.WriteByte('>')
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	_, err := mw.w.Write(b.Bytes())
	return err
}
