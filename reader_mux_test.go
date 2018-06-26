package expect

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReaderMux(t *testing.T) {
	in, out := io.Pipe()
	defer out.Close()
	defer in.Close()

	rm := NewReaderMux(in)
	go rm.Mux()

	tests := []struct {
		title    string
		expected string
	}{
		{
			"Read cancels with deadline",
			"apple",
		},
		{
			"Second read has no bytes stolen",
			"banana",
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			r := rm.Reader(ctx)
			tin, tout := io.Pipe()

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				io.Copy(tout, r)
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := out.Write([]byte(test.expected))
				require.Nil(t, err)
			}()

			for i := 0; i < len(test.expected); i++ {
				p := make([]byte, 1)
				n, err := tin.Read(p)
				require.Nil(t, err)
				require.Equal(t, 1, n)
				require.Equal(t, test.expected[i], p[0])
			}

			cancel()
			wg.Wait()
		})
	}
}
