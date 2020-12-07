package rle

import (
	"strconv"
	"strings"
	"unicode"
)

//  RunLengthEncode RLE кодирование, где последовательность одинаковых символов заменяется на их количество и этот символ
func RunLengthEncode(input string) string {
	notChangedInput := input
	var result strings.Builder
	for len(input) > 0 {
		firstLetter := input[0]
		inputLength := len(input)
		input = strings.TrimLeft(input, string(firstLetter))
		if counter := inputLength - len(input); counter > 1 {
			result.WriteString(strconv.Itoa(counter))
		}
		result.WriteString(string(firstLetter))
		// проверяем что становиться не хуже, если хуже то возвращаем строку без изменений
		if len(result.String()) > len(notChangedInput) {
			return notChangedInput
		}
	}
	result.WriteString("%#%")
	return result.String()
}

// RunLengthDecode метод декодирования RLE, где так жде присуцтвкет проверка на то,
// был ли ипользован RLE
func RunLengthDecode(input string) string {
	last3  := input[len(input)-3:]
	if last3 == "%#%" {
		input = input[:len(input)-3]
		var result strings.Builder
		for len(input) > 0 {
			letterIndex := strings.IndexFunc(input, func(r rune) bool { return !unicode.IsDigit(r) })
			multiply := 1
			if letterIndex != 0 {
				multiply, _ = strconv.Atoi(input[:letterIndex])
			}
			result.WriteString(strings.Repeat(string(input[letterIndex]), multiply))
			input = input[letterIndex+1:]
		}
		return result.String()
	}
	return input
}