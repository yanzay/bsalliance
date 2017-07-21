package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

func roundDuration(d time.Duration) time.Duration {
	return d - (d % time.Second)
}

func pad(first, last string) string {
	fmt.Println(first, last)
	if utf8.RuneCountInString(first) > 16 {
		r := []rune(first)
		r = r[:16]
		first = string(r)
	}
	repeatCount := padWidth - utf8.RuneCountInString(first) - utf8.RuneCountInString(last)
	if repeatCount <= 0 {
		repeatCount = 1
	}
	return first + strings.Repeat(".", repeatCount) + last
}
