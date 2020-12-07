package mtf

import "bytes"

// Алфавит необходимый для работы алгоритма
type SymbolTable string

// Encode метод кодирования где происходит замена каждого входного символа
// его номером в специальном стеке недавно использованных символов.
func (symbols SymbolTable) Encode(s []byte) []byte {
	seq := make([]byte, len(s))
	pad := []byte(symbols)
	for i, c := range s {
		x := bytes.IndexByte(pad, c)
		seq[i] = byte(x)
		copy(pad[1:], pad[:x])
		pad[0] = c
	}
	return seq
}

// Decode метод декодирования
func (symbols SymbolTable) Decode(seq []byte) []byte {
	chars := make([]byte, len(seq))
	pad := []byte(symbols)
	for i, x := range seq {
		c := pad[x]
		chars[i] = c
		copy(pad[1:], pad[:x])
		pad[0] = c
	}
	return chars
}

// AlphabetCreate метод постороения алфавита (уникальных) по входной строке
func AlphabetCreate(input []byte) []byte {
	var res []byte
	for _, b := range input {
		if !bytes.Contains(res, []byte{b}) {
			res = append(res, b)
		}
	}
	return res
}

// GetAlphabet метод получения алфавита (уникальных), так же получаем длину алфавита,
// так как он закодирован в строке по входной строке
func GetAlphabet(input []byte) ([]byte, []byte) {
	num := input[len(input)-1]
	symbols := input[len(input)-int(num)-1:len(input)-1]
	return input[:len(input)-int(num)-1], symbols
}