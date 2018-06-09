package expect

import (
	"bufio"
	"io"
	"testing"
)

// NewTestConsole multiplexes the application's stdout to go's testing logger,
// so that outputs from parallel tests using t.Parallel() is not interleaved.
func NewTestConsole(t *testing.T, opts ...ConsoleOpt) (*Console, error) {
	tf, err := NewTestWriter(t)
	if err != nil {
		return nil, err
	}

	return NewConsole(append(opts, WithStdout(tf))...)
}

// NewTestWriter returns a io.Writer where bytes written to the file are
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
