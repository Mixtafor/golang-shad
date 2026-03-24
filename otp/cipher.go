//go:build !solution

package otp

import (
	"errors"
	"io"
)

//Reader

type cipherReader struct {
	r    io.Reader
	prng io.Reader
}

func (c cipherReader) Read(buf []byte) (int, error) {
	cntin, err := c.r.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return cntin, err
	}

	if err != nil && errors.Is(err, io.EOF) && cntin == 0 {
		return cntin, err
	}

	randomBytes := make([]byte, cntin)
	io.ReadFull(c.prng, randomBytes) // prng doesn't throw exception

	for ind := 0; ind < cntin; ind++ {
		buf[ind] ^= randomBytes[ind]
	}

	return cntin, nil
}

func NewReader(r io.Reader, prng io.Reader) io.Reader {
	return cipherReader{r, prng}
}

//Writer

type cipherWriter struct {
	w         io.Writer
	prng      io.Reader
	modifBuff [256]byte
}

func (c *cipherWriter) Write(buf []byte) (int, error) {
	bufPtr := 0

	for bufPtr < len(buf) {
		copied := copy(c.modifBuff[:], buf[bufPtr:])

		randomBytes := make([]byte, copied)
		io.ReadFull(c.prng, randomBytes)

		for ind := 0; ind < copied; ind++ {
			c.modifBuff[ind] ^= randomBytes[ind]
		}

		cnt, err := c.w.Write(c.modifBuff[:copied])
		if err != nil {
			return bufPtr + cnt, err
		}

		if cnt < copied {
			return bufPtr + cnt, errors.New("error")
		}

		bufPtr += copied
	}
	return bufPtr, nil

}

func NewWriter(w io.Writer, prng io.Reader) io.Writer {
	return &cipherWriter{w: w, prng: prng}
}
