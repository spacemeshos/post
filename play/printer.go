package main

import (
	"strings"
)

func printByte(b byte, msbFirst bool) string {
	//return fmt.Sprintf("%08b",b)

	var sb strings.Builder
	for bit := uint(0); bit < 8; bit++ {
		mask := byte(1 << bit)
		if b&mask == mask {
			sb.WriteString("1")
		} else {
			sb.WriteString("0")
		}
	}

	if msbFirst {
		return Reverse(sb.String())
	}
	return sb.String()
}

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
