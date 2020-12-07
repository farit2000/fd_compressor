package huffman

import (
	"github.com/farit2000/compressor/src/bitio"
	"io"
)

// Writer - это реализация модуля записи Huffman.
// Должен быть закрыт для правильной отправки EOF.
type Writer struct {
	*symbols
	bw *bitio.Writer
}

// NewWriter возвращает новый Writer, используя указанный io.Writer в качестве вывода,
// с параметрами по умолчанию.
func NewWriter(out io.Writer) *Writer {
	return NewWriterOptions(out, nil)
}

//  Читатель сможет только правильно декодировать поток
// создается Writer, если одни и те же параметры используются как в Reader, так и Writer.
func NewWriterOptions(out io.Writer, o *Options) *Writer {
	o = checkOptions(o)
	return &Writer{symbols: newSymbols(o), bw: bitio.NewWriter(out)}
}

// Write записывает сжатую форму p в базовый io.Writer.
// Сжатый байт (байты) не обязательно сбрасывается до закрытия Writer.
func (w *Writer) Write(p []byte) (n int, err error) {
	for i, v := range p {
		if err = w.WriteByte(v); err != nil {
			return i, err
		}
	}
	return len(p), nil
}

// WriteByte записывает сжатую форму b в базовый io.Writer.
// Сжатый байт (байты) не обязательно сбрасывается до закрытия Writer.
func (w *Writer) WriteByte(b byte) (err error) {
	value := ValueType(b)
	node := w.valueMap[value]
	if node == nil {
		// Новое значение, записываем код Хаффмана newValue
		if err = w.bw.WriteBits(w.valueMap[newValue].Code()); err != nil {
			return
		}
		// ... и новое значение
		if err = w.bw.WriteByte(b); err != nil {
			return
		}
		w.insert(value)
	} else {
		// Записываем код Хаффмана узла
		if err = w.bw.WriteBits(node.Code()); err != nil {
			return
		}
		w.update(node)
	}
	return
}

// Close закрывает модуль записи Хаффмана, правильно отправляя EOF.
// Если базовый io.Writer реализует io.Closer,
// он будет закрыт после отправки EOF.
func (w *Writer) Close() (err error) {
	// Если были какие-то данные, выписываем eofValue
	if len(w.leaves) > 2 {
		// Записываем код Хаффмана eofValue
		if err = w.bw.WriteBits(w.valueMap[eofValue].Code()); err != nil {
			return
		}
	}
	return w.bw.Close()
}
