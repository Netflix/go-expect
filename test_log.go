package expect

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"testing"
)

// NewTestConsole returns a new Console that multiplexes the application's
// stdout to go's testing logger. Primarily so that outputs from parallel tests
// using t.Parallel() is not interleaved.
func NewTestConsole(t *testing.T, opts ...ConsoleOpt) (*Console, error) {
	tf, err := NewTestWriter(t)
	if err != nil {
		return nil, err
	}

	return NewConsole(append(opts, WithStdout(tf))...)
}

// NewTestWriter returns an io.Writer where bytes written to the file are
// logged by go's testing logger. Bytes are flushed to the logger on line end.
func NewTestWriter(t *testing.T) (io.Writer, error) {
	r, w := io.Pipe()
	tw := testWriter{t}

	go func() {
		defer r.Close()

		br := bufio.NewReader(r)

		for {
			line, _, err := br.ReadLine()
			if err != nil {
				return
			}

			_, err = tw.Write(line)
			if err != nil {
				return
			}
		}
	}()

	return w, nil
}

// testWriter provides a io.Writer interface to go's testing logger.
type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Log(string(p))
	return len(p), nil
}

// FormatTestLog formats a multiline string by emulating newlines with spaces
// appropriate for number of columns of the output tty, and deleting empty lines
// from end of the string.
//
// Go's test logger adds two newlines for every newline when calling t.Log, so
// we get around this by emulating newlines instead.
func FormatTestLog(out string, cols int) (string, error) {
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return out, nil
	}

	for i := len(lines) - 1; i >= 0; i-- {
		stripped := strings.Replace(lines[i], " ", "", -1)
		if len(stripped) == 0 {
			lines = lines[:len(lines)-1]
		} else {
			break
		}
	}

	var newline []rune
	offset := cols - len(lines[0])
	for i := 0; i < offset; i++ {
		newline = append(newline, ' ')
	}

	out = strings.Join(lines, string(newline))
	return fmt.Sprintf("\n%s", out), nil
}
