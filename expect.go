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
// terminal or console based programs. It is unlike expect and other go
// expect packages in that it does not spawn or control process lifecycle.
// This package only interfaces with a stdin and stdout and controls the
// interaction through those files alone.
package expect

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kr/pty"
)

var (
	// DefaultConsoleOpts is the default configuration for a Console.
	DefaultConsoleOpts = ConsoleOpts{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
	}
)

// ConsoleOpts defines the configuration for a Console.
type ConsoleOpts struct {
	// Stdin is a file that Console will multiplex into the stdin of the program.
	// For example, an user can interact with a password prompt while other
	// prompts are handled by the Console.
	Stdin io.Reader

	// Stdout is a file that Console will write the program output to. It is
	// optional but useful to be in the test output.
	Stdout io.Writer
}

// Console is a controller for interactive applications, allowing automation of
// a terminal or console based program. It parses a given stdin and stdout for
// an expected string, and can send bytes to respond to a match.
type Console struct {
	opts    ConsoleOpts
	ptm     *os.File
	pts     *os.File
	closers []io.Closer
}

// NewConsole creates a new Console with the default options.
func NewConsole() (*Console, error) {
	return NewConsoleWithOpts(DefaultConsoleOpts)
}

// NewConsoleWithOpts creates a new Console with the given options.
func NewConsoleWithOpts(opts ConsoleOpts) (*Console, error) {
	ptm, pts, err := pty.Open()
	if err != nil {
		return nil, err
	}
	closers := append([]io.Closer{}, pts, ptm)

	return &Console{
		opts:    opts,
		ptm:     ptm,
		pts:     pts,
		closers: closers,
	}, nil
}

// Stdin returns a file that Console writes stdin to. Typically this is the
// program's stdin.
func (c *Console) Stdin() *os.File {
	return c.pts
}

// Stdout returns a file that Console reads stdout from. Typically this is the
// program's stdout.
func (c *Console) Stdout() *os.File {
	return c.pts
}

// Close closes the Console's tty, pty and pipe. Calling Close will unblock
// ExpectEOF.
func (c *Console) Close() {
	for _, fd := range c.closers {
		fd.Close()
	}
}

// Expect blocks until it finds the given string from the Console's Stdout
// starting from when Expect was called, and returns the buffer containing the
// match. No extra bytes are read once a match is found.
//
// Expect performs the string search whenever ConsoleOpts.ReadDeadline times
// out before the next byte is read.
func (c *Console) Expect(s string) (string, error) {
	buf := new(bytes.Buffer)
	multi := io.MultiWriter(c.opts.Stdout, buf)

	var content string
	for {
		p := make([]byte, 1)
		n, err := c.ptm.Read(p)
		if err != nil {
			return "", err
		}

		_, err = multi.Write(p[:n])
		if err != nil {
			return "", err
		}

		content = buf.String()
		// Replace with KMP table.
		if strings.Contains(content, s) {
			break
		}
	}

	return content, nil
}

// ExpectEOF blocks until an EOF is read or an error occurs. It returns the
// number of bytes copied and the first error encountered, if any.
func (c *Console) ExpectEOF() (int64, error) {
	return io.Copy(c.opts.Stdout, c.ptm)
}

// Send writes the given string to the Console's Stdin.
func (c *Console) Send(s string) (int, error) {
	return c.ptm.WriteString(s)
}

// SendLine writes the given string with a newline to the Console's Stdin.
func (c *Console) SendLine(s string) (int, error) {
	return c.Send(fmt.Sprintf("%s\n", s))
}
