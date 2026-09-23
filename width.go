package main

//go:generate go run gen_width.go

import (
	"sort"
	"unicode"
)

// displayWidth returns the terminal columns of s, ANSI stripped.
func displayWidth(s string) int {
	w := 0
	for _, r := range ansi.ReplaceAllString(s, "") {
		if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
			continue
		}
		// Ambiguous counts as 1: every box-drawing and block glyph is
		// Ambiguous, and WT/iTerm2/Linux terminals render them narrow.
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func isWide(r rune) bool {
	i := sort.Search(len(wideRanges), func(i int) bool { return wideRanges[i][1] >= r })
	return i < len(wideRanges) && wideRanges[i][0] <= r
}
