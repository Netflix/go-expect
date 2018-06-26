package expect

import (
	"context"
	"fmt"
	"io"
)

type ReaderMux struct {
	reader io.Reader
	bytec  chan byte
}

func NewReaderMux(reader io.Reader) *ReaderMux {
	return &ReaderMux{
		reader: reader,
		bytec:  make(chan byte),
	}
}

func (rm *ReaderMux) Mux() error {
	for {
		p := make([]byte, 1)
		n, err := rm.reader.Read(p)
		if err != nil {
			return err
		}
		if n == 0 {
			panic("non eof read 0 bytes")
		}

		rm.bytec <- p[0]
	}
	return nil
}

func (rm *ReaderMux) Reader(ctx context.Context) io.Reader {
	return NewChanReader(ctx, rm.bytec)
}

type chanReader struct {
	ctx   context.Context
	bytec <-chan byte
}

func NewChanReader(ctx context.Context, bytec <-chan byte) io.Reader {
	return &chanReader{
		ctx:   ctx,
		bytec: bytec,
	}
}

func (cr *chanReader) Read(p []byte) (n int, err error) {
	select {
	case <-cr.ctx.Done():
		return 0, io.EOF
	case b := <-cr.bytec:
		if len(p) < 1 {
			return 0, fmt.Errorf("cannot read into 0 len byte slice")
		}
		p[0] = b
		return 1, nil
	}
}
