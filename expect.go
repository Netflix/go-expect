// Copyright 2018 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package expect provides an expect-like interface to automate control of
// applications. It is unlike expect in that it does not spawn or manage
// process lifecycle. This package only focuses on expecting output and sending
// input through it's psuedoterminal.
package expect

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"unicode/utf8"
)

// Expect reads from Console's tty until s is encountered and returns the
// buffer read by Console. No extra bytes are read once a match is found, so if
// a program isn't expecting input yet it will be blocked. Sends are queued up
// so the next Expect will read the remaining bytes (i.e. rest of prompt) or
// ExpectEOF if nothing else is expected.
func (c *Console) Expect(s string) (string, error) {
	buf := new(bytes.Buffer)
	writer := io.MultiWriter(append(c.opts.Stdouts, buf)...)
	runeWriter := bufio.NewWriterSize(writer, utf8.UTFMax)

	var content string
	for {
		content = buf.String()
		r, _, err := c.runeReader.ReadRune()
		if err != nil {
			return content, err
		}

		c.Logf("expect read: %q", string(r))
		_, err = runeWriter.WriteRune(r)
		if err != nil {
			return content, err
		}

		// Immediately flush rune to the underlying writers.
		err = runeWriter.Flush()
		if err != nil {
			return content, err
		}

		content = buf.String()
		// Replace with KMP table.
		if strings.Contains(content, s) {
			break
		}
	}

	return content, nil
}

// ExpectEOF reads out the Console's tty until EOF or an error occurs. If
// Console has multiple stdouts, the bytes read from the tty are written to all
// stdouts.
func (c *Console) ExpectEOF() (int, error) {
	if len(c.opts.Stdouts) == 0 {
		return c.copyWithLog(ioutil.Discard, c.runeReader)
	}

	w := io.MultiWriter(c.opts.Stdouts...)
	return c.copyWithLog(w, c.runeReader)
}

func (c *Console) copyWithLog(w io.Writer, r io.Reader) (int, error) {
	for {
		p := make([]byte, 1)
		n, err := r.Read(p)
		if err != nil {
			return n, err
		}
		c.Logf("expect eof read: %q", p)

		n, err = w.Write(p[:n])
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
}
