package bitio

import (
	"bufio"
	"io"
)

type readerAndByteReader interface {
	io.Reader
	io.ByteReader
}

type Reader struct {
	in    readerAndByteReader
	cache byte // здесь хранятся непрочитанные биты
	bits  byte // количество непрочитанных битов в кеше
	// TryError содержит первую ошибку, возникшую в методах TryXXX ().
	TryError error
}

// NewReader возвращает новый Reader, используя указанный io.Reader в качестве входа (источника).
func NewReader(in io.Reader) *Reader {
	bin, ok := in.(readerAndByteReader)
	if !ok {
		bin = bufio.NewReader(in)
	}
	return &Reader{in: bin}
}

// Read реализует io.Reader и дает представление битового потока на уровне байтов.
// Это даст лучшую производительность, если базовый io.Reader выровнен
// до границы байта (иначе все отдельные байты собираются из нескольких байтов).
// Границу байта можно обеспечить, вызвав Align ().
func (r *Reader) Read(p []byte) (n int, err error) {
	// r.bits будет таким же после чтения 8 бит, поэтому нам не нужно это обновлять.
	if r.bits == 0 {
		return r.in.Read(p)
	}
	for ; n < len(p); n++ {
		if p[n], err = r.readUnalignedByte(); err != nil {
			return
		}
	}
	return
}

// ReadBits считывает n битов и возвращает их как младшие n битов u.
func (r *Reader) ReadBits(n uint8) (u uint64, err error) {
	// Some optimization, frequent cases
	if n < r.bits {
		// в кеше есть все необходимые биты, и есть некоторые лишние, которые останутся в кеше
		shift := r.bits - n
		u = uint64(r.cache >> shift)
		r.cache &= 1<<shift - 1
		r.bits = shift
		return
	}

	if n > r.bits {
		// нужны все биты кеша, и этого недостаточно, поэтому будет прочитано больше
		if r.bits > 0 {
			u = uint64(r.cache)
			n -= r.bits
		}
		// Читаем целые байты
		for n >= 8 {
			b, err2 := r.in.ReadByte()
			if err2 != nil {
				return 0, err2
			}
			u = u<<8 + uint64(b)
			n -= 8
		}
		// Считываем последнюю фракицю, если есть
		if n > 0 {
			if r.cache, err = r.in.ReadByte(); err != nil {
				return 0, err
			}
			shift := 8 - n
			u = u<<n + uint64(r.cache>>shift)
			r.cache &= 1<<shift - 1
			r.bits = shift
		} else {
			r.bits = 0
		}
		return u, nil
	}

	// в кеше ровно столько, сколько нужно
	r.bits = 0 // не нужно очищать кеш, будет перезаписано при следующем чтении
	return uint64(r.cache), nil
}

// ReadByte реализует io.ByteReader.
func (r *Reader) ReadByte() (b byte, err error) {
	// r.bits будет таким же после чтения 8 бит, поэтому нам не нужно это обновлять.
	if r.bits == 0 {
		return r.in.ReadByte()
	}
	return r.readUnalignedByte()
}

// readUnalignedByte считывает следующие 8 бит, которые (могут быть) невыровненными, и возвращает их как байт.
func (r *Reader) readUnalignedByte() (b byte, err error) {
	// r.bits будет таким же после чтения 8 бит, поэтому нам не нужно это обновлять.
	bits := r.bits
	b = r.cache << (8 - bits)
	r.cache, err = r.in.ReadByte()
	if err != nil {
		return 0, err
	}
	b |= r.cache >> bits
	r.cache &= 1<<bits - 1
	return
}

// ReadBool читает следующий бит и возвращает истину, если он равен 1.
func (r *Reader) ReadBool() (b bool, err error) {
	if r.bits == 0 {
		r.cache, err = r.in.ReadByte()
		if err != nil {
			return
		}
		b = (r.cache & 0x80) != 0
		r.cache, r.bits = r.cache&0x7f, 7
		return
	}
	r.bits--
	b = (r.cache & (1 << r.bits)) != 0
	r.cache &= 1<<r.bits - 1
	return
}

// Align выравнивает битовый поток по границе байта,
// поэтому следующее чтение будет читать / использовать данные из следующего байта.
// Возвращает количество непрочитанных / пропущенных бит.
func (r *Reader) Align() (skipped uint8) {
	skipped = r.bits
	r.bits = 0 // no need to clear cache, will be overwritten on next read
	return
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает Read (),
// возвращает предоставленные данные и сохраняет ошибку в поле TryError.
func (r *Reader) TryRead(p []byte) (n int) {
	if r.TryError == nil {
		n, r.TryError = r.Read(p)
	}
	return
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает ReadBits (),
// возвращает предоставленные данные и сохраняет ошибку в поле TryError.
func (r *Reader) TryReadBits(n uint8) (u uint64) {
	if r.TryError == nil {
		u, r.TryError = r.ReadBits(n)
	}
	return
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает ReadByte (),
// возвращает предоставленные данные и сохраняет ошибку в поле TryError.
func (r *Reader) TryReadByte() (b byte) {
	if r.TryError == nil {
		b, r.TryError = r.ReadByte()
	}
	return
}

// Если была предыдущая ошибка TryError, она ничего не делает. В противном случае он вызывает ReadBool (),
// возвращает предоставленные данные и сохраняет ошибку в поле TryError.
func (r *Reader) TryReadBool() (b bool) {
	if r.TryError == nil {
		b, r.TryError = r.ReadBool()
	}
	return
}
