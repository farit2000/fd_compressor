package bwt

import (
	"errors"
	"fmt"
)

const (
	_BWTS_MAX_BLOCK_SIZE = 1024 * 1024 * 1024 // 1 GB
)

// Биективная версия преобразования Барроуза-Уиллера BWTS https://ru.qaz.wiki/wiki/Burrows–Wheeler_transform
// Основное преимущество перед обычным BWT в том, что нет необходимости в первичном
// индексе (отсюда биективность). BWTS примерно на 10% медленнее, чем BWT.
// алгоритм сортировки использован готовый,
// так как только при нем деградация скорости при большом наборе данных наименьший
type BWTS struct {
	buffer1 []int32
	buffer2 []int32
	saAlgo  *DivSufSort
}

// NewBWTS создает новый экземпляр BWTS
func NewBWTS() (*BWTS, error) {
	this := &BWTS{}
	this.buffer1 = make([]int32, 0)
	this.buffer2 = make([]int32, 0)
	return this, nil
}

// Вперед применяет функцию к src и записывает результат
// в пункт назначения. Возвращает количество прочитанных байтов, количество байтов.
// written and possibly an error.
func (this *BWTS) Forward(src, dst []byte) (uint, uint, error) {
	if len(src) == 0 {
		return 0, 0, nil
	}
	if &src[0] == &dst[0] {
		return 0, 0, errors.New("Input and output buffers cannot be equal")
	}
	count := len(src)
	count32 := int32(count)
	if count > MaxBWTSBlockSize() {
		// Неустранимая ошибка: вместо того, чтобы молча прервать преобразование,
		// выдаем фатальную ошибку.
		errMsg := fmt.Sprintf("The max BWTS block size is %v, got %v", MaxBWTSBlockSize(), count)
		panic(errors.New(errMsg))
	}
	if count > len(dst) {
		errMsg := fmt.Sprintf("Block size is %v, output buffer length is %v", count, len(dst))
		return 0, 0, errors.New(errMsg)
	}
	if count < 2 {
		if count == 1 {
			dst[0] = src[0]
		}
		return uint(count), uint(count), nil
	}
	if this.saAlgo == nil {
		var err error
		if this.saAlgo, err = NewDivSufSort(); err != nil {
			return 0, 0, err
		}
	}
	// Ленивое распределение динамической памяти
	if len(this.buffer1) < count {
		this.buffer1 = make([]int32, count)
	}
	if len(this.buffer2) < count {
		this.buffer2 = make([]int32, count)
	}
	// Псевдоним
	sa := this.buffer1[0:count]
	isa := this.buffer2[0:count]
	this.saAlgo.ComputeSuffixArray(src[0:count], sa)
	for i := range isa {
		isa[sa[i]] = int32(i)
	}
	min := isa[0]
	idxMin := int32(0)
	for i := int32(1); i < count32 && min > 0; i++ {
		if isa[i] >= min {
			continue
		}
		refRank := this.moveLyndonWordHead(sa, isa, src, count32, idxMin, i-idxMin, min)
		for j := i - 1; j > idxMin; j-- {
			// перебираем новое слово Lyndon от конца до начала
			testRank := isa[j]
			startRank := testRank
			for testRank < count32-1 {
				nextRankStart := sa[testRank+1]

				if j > nextRankStart || src[j] != src[nextRankStart] || refRank < isa[nextRankStart+1] {
					break
				}
				sa[testRank] = nextRankStart
				isa[nextRankStart] = testRank
				testRank++
			}
			sa[testRank] = int32(j)
			isa[j] = testRank
			refRank = testRank
			if startRank == testRank {
				break
			}
		}
		min = isa[i]
		idxMin = i
	}
	min = count32
	for i := 0; i < count; i++ {
		if isa[i] >= min {
			dst[isa[i]] = src[i-1]
			continue
		}
		if min < count32 {
			dst[min] = src[i-1]
		}
		min = isa[i]
	}
	dst[0] = src[count-1]
	return uint(count), uint(count), nil
}

func (this *BWTS) moveLyndonWordHead(sa, isa []int32, data []byte, count, start, size, rank int32) int32 {
	end := start + size
	for rank+1 < count {
		nextStart0 := sa[rank+1]
		if nextStart0 <= end {
			break
		}
		nextStart := nextStart0
		k := int32(0)
		for k < size && nextStart < count && data[start+k] == data[nextStart] {
			k++
			nextStart++
		}
		if k == size && rank < isa[nextStart] {
			break
		}
		if k < size && nextStart < count && data[start+k] < data[nextStart] {
			break
		}
		sa[rank] = nextStart0
		isa[nextStart0] = rank
		rank++
	}
	sa[rank] = start
	isa[start] = rank
	return rank
}

// Inverse применяет обратную функцию к src и записывает результат
// в пункт назначения. Возвращает количество прочитанных байтов, количество байтов.
// записано и возможно ошибка.
func (this *BWTS) Inverse(src, dst []byte) (uint, uint, error) {
	if len(src) == 0 {
		return 0, 0, nil
	}
	if &src[0] == &dst[0] {
		return 0, 0, errors.New("Input and output buffers cannot be equal")
	}
	count := len(src)
	if count > MaxBWTSBlockSize() {
		// Неустранимая ошибка: вместо того, чтобы молча прервать преобразование,
		// выдаем фатальную ошибку.
		errMsg := fmt.Sprintf("The max BWTS block size is %v, got %v", MaxBWTSBlockSize(), count)
		panic(errors.New(errMsg))
	}
	if count > len(dst) {
		errMsg := fmt.Sprintf("Block size is %v, output buffer length is %v", count, len(dst))
		return 0, 0, errors.New(errMsg)
	}
	if count < 2 {
		if count == 1 {
			dst[0] = src[0]
		}
		return uint(count), uint(count), nil
	}
	// Ленивое распределение динамической памяти
	if len(this.buffer1) < count {
		this.buffer1 = make([]int32, count)
	}
	lf := this.buffer1
	buckets := [256]int32{}
	for i := 0; i < count; i++ {
		buckets[src[i]]++
	}
	sum := int32(0)
	for i := range &buckets {
		sum += buckets[i]
		buckets[i] = sum - buckets[i]
	}
	for i := 0; i < count; i++ {
		lf[i] = buckets[src[i]]
		buckets[src[i]]++
	}
	// Строим инверсию
	for i, j := 0, count-1; j >= 0; i++ {
		if lf[i] < 0 {
			continue
		}
		p := int32(i)
		for {
			dst[j] = src[p]
			j--
			t := lf[p]
			lf[p] = -1
			p = t
			if lf[p] < 0 {
				break
			}
		}
	}
	return uint(count), uint(count), nil
}

// MaxBWTSBlockSize возвращает максимальный размер блока для преобразования
func MaxBWTSBlockSize() int {
	return _BWTS_MAX_BLOCK_SIZE
}
