package bitio

import (
	"bufio"
	"io"
)

type writerAndByteWriter interface {
	io.Writer
	io.ByteWriter
}

type Writer struct {
	out       writerAndByteWriter
	wrapperbw *bufio.Writer // оболочка bufio.Writer, если цель не реализует io.ByteWriter
	cache     byte          // здесь хранятся незаписанные биты
	bits      byte         // количество незаписанных битов в кеше
	// TryError содержит первую ошибку, возникшую в методах TryXXX ().
	TryError error
}

// Должен быть закрыт, чтобы очистить кешированные данные.
// Если не можем его закрывать, можно также принудительно очистить данные
// вызовом Align().
func NewWriter(out io.Writer) *Writer {
	w := &Writer{}
	var ok bool
	w.out, ok = out.(writerAndByteWriter)
	if !ok {
		w.wrapperbw = bufio.NewWriter(out)
		w.out = w.wrapperbw
	}
	return w
}

// Запись реализует io.Writer и предоставляет байтовый интерфейс для битового потока.
// Это даст лучшую производительность, если базовый io.Writer выровнен
// до границы байта (иначе все отдельные байты распределяются на несколько байтов).
// Границу байта можно обеспечить, вызвав Align ().
func (w *Writer) Write(p []byte) (n int, err error) {
	// w.bits будет таким же после записи 8 бит, поэтому нам не нужно это обновлять.
	if w.bits == 0 {
		return w.out.Write(p)
	}
	for i, b := range p {
		if err = w.writeUnalignedByte(b); err != nil {
			return i, err
		}
	}
	return len(p), nil
}

// WriteBits записывает n младших битов r.
func (w *Writer) WriteBits(r uint64, n uint8) (err error) {
	// если бы r имел биты, установленные в n или более высоких позициях (с нулевым индексом),
	// Реализация WriteBitsUnsafe может "повредить" биты в кеше.
	return w.WriteBitsUnsafe((r & (1<<n - 1)), n)
}

// WriteBitsUnsafe записывает n младших битов r.
func (w *Writer) WriteBitsUnsafe(r uint64, n uint8) (err error) {
	newbits := w.bits + n
	if newbits < 8 {
		// r помещается в кеш, запись в out не произойдет
		w.cache |= byte(r) << (8 - newbits)
		w.bits = newbits
		return nil
	}
	if newbits > 8 {
		// кеш будет заполнен, и будет больше бит для записи
		// "Заполняем кеш" и записываем его
		free := 8 - w.bits
		err = w.out.WriteByte(w.cache | byte(r>>(n-free)))
		if err != nil {
			return
		}
		n -= free
		// записываем целые байты
		for n >= 8 {
			n -= 8
			// Нет необходимости маскировать r, преобразование в байты замаскирует более высокие биты
			err = w.out.WriteByte(byte(r >> n))
			if err != nil {
				return
			}
		}
		// Помещаем оставшееся в кеш
		if n > 0 {
			// Примечание: n <8 (в случае n = 8, 1 << n приведет к переполнению байта)
			w.cache, w.bits = (byte(r)&((1<<n)-1))<<(8-n), n
		} else {
			w.cache, w.bits = 0, 0
		}
		return nil
	}
	// кэш будет заполнен ровно теми битами, которые нужно записать
	bb := w.cache | byte(r)
	w.cache, w.bits = 0, 0
	return w.out.WriteByte(bb)
}

// WriteByte реализует io.ByteWriter.
func (w *Writer) WriteByte(b byte) (err error) {
	// w.bits будет таким же после записи 8 бит, поэтому нам не нужно это обновлять.
	if w.bits == 0 {
		return w.out.WriteByte(b)
	}
	return w.writeUnalignedByte(b)
}

// writeUnalignedByte записывает 8 бит, которые (могут быть) невыровненными.
func (w *Writer) writeUnalignedByte(b byte) (err error) {
	// w.bits будет таким же после записи 8 бит, поэтому нам не нужно это обновлять.
	bits := w.bits
	err = w.out.WriteByte(w.cache | b>>bits)
	if err != nil {
		return
	}
	w.cache = (b & (1<<bits - 1)) << (8 - bits)
	return
}

// WriteBool записывает один бит: 1, если параметр равен true, в противном случае - 0.
func (w *Writer) WriteBool(b bool) (err error) {
	if w.bits == 7 {
		if b {
			err = w.out.WriteByte(w.cache | 1)
		} else {
			err = w.out.WriteByte(w.cache)
		}
		if err != nil {
			return
		}
		w.cache, w.bits = 0, 0
		return nil
	}
	w.bits++
	if b {
		w.cache |= 1 << (8 - w.bits)
	}
	return nil
}

// Align выравнивает битовый поток по границе байта,
// так что следующая запись начнется / перейдет в новый байт.
// Если есть кешированные биты, они сначала записываются в вывод.
// Возвращает количество пропущенных (не установленных, но все еще записанных) битов.
func (w *Writer) Align() (skipped uint8, err error) {
	if w.bits > 0 {
		if err = w.out.WriteByte(w.cache); err != nil {
			return
		}

		skipped = 8 - w.bits
		w.cache, w.bits = 0, 0
	}
	if w.wrapperbw != nil {
		err = w.wrapperbw.Flush()
	}
	return
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает Write (),
// возвращает предоставленные данные и сохраняет ошибку в поле TryError.
func (w *Writer) TryWrite(p []byte) (n int) {
	if w.TryError == nil {
		n, w.TryError = w.Write(p)
	}
	return
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает WriteBits (),
// и сохраняет ошибку в поле TryError.
func (w *Writer) TryWriteBits(r uint64, n uint8) {
	if w.TryError == nil {
		w.TryError = w.WriteBits(r, n)
	}
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает WriteBitsUnsafe (),
// и сохраняет ошибку в поле TryError.
func (w *Writer) TryWriteBitsUnsafe(r uint64, n uint8) {
	if w.TryError == nil {
		w.TryError = w.WriteBitsUnsafe(r, n)
	}
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает WriteByte (),
// и сохраняет ошибку в поле TryError.
func (w *Writer) TryWriteByte(b byte) {
	if w.TryError == nil {
		w.TryError = w.WriteByte(b)
	}
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает WriteBool (),
// и сохраняет ошибку в поле TryError.
func (w *Writer) TryWriteBool(b bool) {
	if w.TryError == nil {
		w.TryError = w.WriteBool(b)
	}
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает Align (),
// возвращает предоставленные данные и сохраняет ошибку в поле TryError.
func (w *Writer) TryAlign() (skipped uint8) {
	if w.TryError == nil {
		skipped, w.TryError = w.Align()
	}
	return
}

// Close реализует io.Closer.
func (w *Writer) Close() (err error) {
	// Make sure cached bits are flushed:
	if _, err = w.Align(); err != nil {
		return
	}
	return nil
}
