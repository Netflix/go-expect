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
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// ExpectOpt allows setting Expect options.
type ExpectOpt func(*ExpectOpts) error

// ExpectOpts provides additional options on Expect.
type ExpectOpts struct {
	Timeout time.Duration
}

// WithTimeout sets the deadline for Expect.
// A zero value for timouet means Read will not time out.
//
// An error returned after a timeout fails will implement the
// Timeout method, and calling the Timeout method will return true.
// The PathError and SyscallError types implement the Timeout method.
// In general, call IsTimeout to test whether an error indicates a timeout.
func WithTimeout(timeout time.Duration) ExpectOpt {
	return func(opts *ExpectOpts) error {
		opts.Timeout = timeout
		return nil
	}
}

// Expect reads from Console's tty until s is encountered and returns the
// buffer read by Console. No extra bytes are read once a match is found, so if
// a program isn't expecting input yet it will be blocked. Sends are queued up
// so the next Expect will read the remaining bytes (i.e. rest of prompt) or
// ExpectEOF if nothing else is expected.
func (c *Console) Expect(s string, opts ...ExpectOpt) (string, error) {
	var options ExpectOpts
	for _, opt := range opts {
		if err := opt(&options); err != nil {
			return "", err
		}
	}

	var reader io.ReadCloser
	if options.Timeout == 0 {
		reader = c.ptm
	} else {
		var err error
		reader, err = readerWithDeadline(c.ptm, options.Timeout)
		if err != nil {
			return "", err
		}
		defer reader.Close()
	}

	buf := new(bytes.Buffer)
	runeReader := bufio.NewReaderSize(reader, utf8.UTFMax)
	writer := io.MultiWriter(append(c.opts.Stdouts, buf)...)
	runeWriter := bufio.NewWriterSize(writer, utf8.UTFMax)

	var content string
	for {
		content = buf.String()
		r, _, err := runeReader.ReadRune()
		if err != nil {
			return content, err
		}

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

func readerWithDeadline(r io.Reader, timeout time.Duration) (io.ReadCloser, error) {
	rp, wp, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	err = rp.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	go func() {
		io.Copy(wp, r)
	}()

	return rp, nil
}

// ExpectEOF reads out the Console's tty until EOF or an error occurs. If
// Console has multiple stdouts, the bytes read from the tty are written to all
// stdouts.
func (c *Console) ExpectEOF(opts ...ExpectOpt) (int64, error) {
	var options ExpectOpts
	for _, opt := range opts {
		if err := opt(&options); err != nil {
			return 0, err
		}
	}

	var r io.ReadCloser
	if options.Timeout == 0 {
		r = c.ptm
	} else {
		var err error
		r, err = readerWithDeadline(c.ptm, options.Timeout)
		if err != nil {
			return 0, err
		}
		defer r.Close()
	}

	if len(c.opts.Stdouts) == 0 {
		return io.Copy(ioutil.Discard, r)
	}

	w := io.MultiWriter(c.opts.Stdouts...)
	return io.Copy(w, r)
}
