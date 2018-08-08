package expect

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPassthroughPipe(t *testing.T) {
	r, w := io.Pipe()

	passthroughPipe, err := NewPassthroughPipe(r)
	require.NoError(t, err)

	err = passthroughPipe.SetReadDeadline(time.Now().Add(time.Hour))
	require.NoError(t, err)

	pipeError := errors.New("pipe error")
	err = w.CloseWithError(pipeError)
	require.NoError(t, err)

	p := make([]byte, 1)
	_, err = passthroughPipe.Read(p)
	require.Equal(t, err, pipeError)
}

func TestPassthroughPipeTimeout(t *testing.T) {
	r, w := io.Pipe()

	passthroughPipe, err := NewPassthroughPipe(r)
	require.NoError(t, err)

	err = passthroughPipe.SetReadDeadline(time.Now())
	require.NoError(t, err)

	_, err = w.Write([]byte("gibberish"))
	require.NoError(t, err)

	p := make([]byte, 1)
	_, err = passthroughPipe.Read(p)
	require.True(t, os.IsTimeout(err))
}
