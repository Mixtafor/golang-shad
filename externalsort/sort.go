//go:build !solution

package externalsort

import (
	"container/heap"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"unicode/utf8"
	"unsafe"
)

type LineReaderT struct {
	r        io.Reader
	buf      [256]byte
	offset   int
	cntBytes int
	splitSym rune
}

func ParseLine(b []byte, splitSym rune) (res string, hasSplitSym bool) {
	if len(b) == 0 {
		return "", false
	}

	i := 0
	for i < len(b) {
		r, size := utf8.DecodeRune(b[i:])
		if r == splitSym {
			return unsafe.String(&b[0], i), true
		}
		i += size
	}

	return unsafe.String(&b[0], len(b)), false
}

func isValidRead(str string, err error) (bool, error) {
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	if errors.Is(err, io.EOF) && len(str) == 0 {
		return false, err
	}

	return true, nil
}

func (l *LineReaderT) ReadLine() (string, error) {
	sb := strings.Builder{}
	buf := make([]byte, 0)

	if l.offset < l.cntBytes {
		buf = l.buf[l.offset:l.cntBytes]
	} else {
		cnt, err := l.r.Read(l.buf[:])
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}

		buf = l.buf[:cnt]
		l.cntBytes = cnt
		l.offset = 0

		if cnt == 0 && errors.Is(err, io.EOF) {
			return sb.String(), io.EOF
		}
	}

	for { // todo inf cycle
		res, hasSplitSym := ParseLine(buf, l.splitSym)
		sb.WriteString(res)

		if hasSplitSym {
			l.offset += len(res) + utf8.RuneLen(l.splitSym)
			return sb.String(), nil
		}

		cnt, err := l.r.Read(l.buf[:])
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}

		buf = l.buf[:cnt]
		l.cntBytes = cnt
		l.offset = 0

		if cnt == 0 && errors.Is(err, io.EOF) {
			return sb.String(), io.EOF
		}

	}

	return sb.String(), io.EOF
}

func NewReader(r io.Reader) LineReader {
	return &LineReaderT{r: r, splitSym: '\n'}
}

type LineWriterT struct {
	w io.Writer
}

func (lw *LineWriterT) Write(str string) error {
	_, err := lw.w.Write(unsafe.Slice(unsafe.StringData(str), len(str)))
	if err != nil {
		return err
	}

	_, err = lw.w.Write([]byte("\n"))
	if err != nil {
		return err
	}
	return nil
}

func NewWriter(w io.Writer) LineWriter {
	return &LineWriterT{w: w}
}

type heapNode struct {
	str           string
	lineReaderInd int
}

type strHeap []heapNode

func (h strHeap) Len() int           { return len(h) }
func (h strHeap) Less(i, j int) bool { return h[i].str < h[j].str }
func (h strHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *strHeap) Push(x any) {
	typedX, ok := x.(heapNode)
	if !ok {
		panic("strHeap contains not heapNode type")
	}

	*h = append(*h, typedX)
}

func (h *strHeap) Pop() any {
	n := len(*h)
	x := (*h)[n-1]
	*h = (*h)[0 : n-1]
	return x
}

func Merge(w LineWriter, readers ...LineReader) error {
	strheap := strHeap{}
	heap.Init(&strheap)

	for i, reader := range readers {
		str, err := reader.ReadLine()
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if errors.Is(err, io.EOF) && len(str) == 0 {
			continue
		}

		heap.Push(&strheap, heapNode{str: str, lineReaderInd: i})
	}

	for strheap.Len() != 0 {
		top := heap.Pop(&strheap)
		typedTop, ok := top.(heapNode)

		if !ok {
			panic("strHeap contains not heapNode type")
		}

		err := w.Write(typedTop.str)
		if err != nil {
			return err
		}

		str, err := readers[typedTop.lineReaderInd].ReadLine()
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if errors.Is(err, io.EOF) && len(str) == 0 {
			continue
		}

		heap.Push(&strheap, heapNode{str: str, lineReaderInd: typedTop.lineReaderInd})
	}

	return nil
}

func Sort(w io.Writer, in ...string) error {
	readers := make([]LineReader, len(in))

	for i, filePath := range in {
		filePtr, err := os.OpenFile(filePath, os.O_RDWR, 0)
		if err != nil {
			return err
		}

		defer filePtr.Close()
		lr := NewReader(filePtr)

		sorted := make([]string, 0)
		for {
			str, err := lr.ReadLine()
			if err != nil && !errors.Is(err, io.EOF) {
				return err
			}
			if errors.Is(err, io.EOF) && len(str) == 0 {
				break
			}

			sorted = append(sorted, str)
		}

		slices.Sort(sorted)
		_, err = filePtr.Seek(0, 0)
		if err != nil {
			return err
		}

		err = filePtr.Truncate(0)
		if err != nil {
			return err
		}

		lw := NewWriter(filePtr)

		for _, str := range sorted {
			err := lw.Write(str)
			if err != nil {
				return err
			}
		}

		_, err = filePtr.Seek(0, 0)
		if err != nil {
			return err
		}

		readers[i] = NewReader(filePtr)
	}

	err := Merge(NewWriter(w), readers...)
	if err != nil {
		return err
	}

	return nil
}
