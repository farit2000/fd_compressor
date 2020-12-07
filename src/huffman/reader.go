package huffman

import (
	"github.com/farit2000/compressor/src/bitio"
	"io"
)

// Reader - это реализация считывателя Хаффмана.
// Он также реализует io.ByteReader.
type Reader struct {
	*symbols
	br *bitio.Reader
}

// NewReader возвращает новый Reader, используя указанный io.Reader в качестве ввода (источника),
// с параметрами по умолчанию.
func NewReader(in io.Reader) *Reader {
	return NewReaderOptions(in, nil)
}

// NewReaderOptions возвращает новый Reader, используя указанный io.Reader в качестве входа (источника)
// с указанными опциями.
func NewReaderOptions(in io.Reader, o *Options) *Reader {
	o = checkOptions(o)
	return &Reader{symbols: newSymbols(o), br: bitio.NewReader(in)}
}

// Чтение распаковывает до len (p) байтов из источника.
func (r *Reader) Read(p []byte) (n int, err error) {
	for i := range p {
		if p[i], err = r.ReadByte(); err != nil {
			return i, err
		}
	}
	return len(p), nil
}

// ReadByte распаковывает один байт.
func (r *Reader) ReadByte() (b byte, err error) {
	// Read Huffman code
	br := r.br
	node := r.root
	for node.Left != nil { // читаем, пока не дойдем до листа
		var right bool
		if right, err = br.ReadBool(); err != nil {
			return
		} else if right {
			node = node.Right
		} else {
			node = node.Left
		}
	}
	switch node.Value {
	case newValue:
		if b, err = br.ReadByte(); err != nil {
			return
		}
		r.insert(ValueType(b))
		return
	case eofValue:
		return 0, io.EOF
	default:
		r.update(node)
		return byte(node.Value), nil
	}
}
