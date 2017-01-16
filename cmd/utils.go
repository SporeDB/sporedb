package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/chzyer/readline"
)

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func read(s string, d string) string {
	l, err := readline.Line(s + " [" + d + "]: ")
	check(err)

	if l == "" {
		l = d
	}

	return l
}

func readInt(s string, d int) int {
	for {
		l := read(s, strconv.Itoa(d))
		n, err := strconv.Atoi(l)
		if err == nil {
			return n
		}
	}
}
