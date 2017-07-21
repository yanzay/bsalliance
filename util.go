package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const padWidth = 23

func roundDuration(d time.Duration) time.Duration {
	return d - (d % time.Second)
}

func pad(first, last string) string {
	fmt.Println(first, last)
	firstLen := utf8.RuneCountInString(first)
	lastLen := utf8.RuneCountInString(last)
	if firstLen+lastLen > padWidth-1 {
		r := []rune(first)
		r = r[:padWidth-lastLen-1]
		first = string(r)
		firstLen = utf8.RuneCountInString(first)
	}
	repeatCount := padWidth - firstLen - lastLen
	return first + strings.Repeat(".", repeatCount) + last
}
