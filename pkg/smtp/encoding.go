package smtp

import (
	"encoding/base64"
	"io"
)

type splitLineWriter struct {
	w io.Writer

	lineLength int
	max        int
	sep        []byte
}

func newSplitLineWriter(w io.Writer, maxLineLength int, sep []byte) *splitLineWriter {
	return &splitLineWriter{
		w:   w,
		max: maxLineLength,
		sep: sep,
	}
}

func (l *splitLineWriter) Write(p []byte) (n int, err error) {
	remain := len(p)

	for remain > 0 {
		if l.lineLength >= l.max {
			_, err = l.w.Write(l.sep)
			if err != nil {
				return
			}

			l.lineLength = 0
		}

		batch := min(l.max-l.lineLength, remain)

		var proceeded int
		proceeded, err = l.w.Write(p[n : n+batch])
		if err != nil {
			n += proceeded
			return
		}

		l.lineLength += batch
		n += batch
		remain -= batch
	}

	return
}

type Base64Encoder struct {
	w io.WriteCloser
}

func NewBase64Encoder(w io.Writer) *Base64Encoder {
	// RFC2045 requires:
	// The encoded output stream must be represented in lines of no more
	// than 76 characters each.
	//
	// https://www.ietf.org/rfc/rfc2045.txt#:~:text=The%20encoded%20output%20stream%20must%20be%20represented%20in%20lines%20of%20no%20more%0A%20%20%20than%2076%20characters%20each.
	splitted := newSplitLineWriter(w, 76, []byte("\r\n"))
	return &Base64Encoder{
		w: base64.NewEncoder(base64.StdEncoding, splitted),
	}
}

func (b *Base64Encoder) Write(p []byte) (n int, err error) {
	return b.w.Write(p)
}

func (b *Base64Encoder) Close() error {
	return b.w.Close()
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
