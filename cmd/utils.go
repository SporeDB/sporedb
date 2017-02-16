package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
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

func readBool(s string, d bool) bool {
	s = s + " [y/n]"
	ds := "n"
	if d {
		ds = "y"
	}

	l := read(s, ds)
	if strings.ToLower(l) == "y" {
		return true
	}
	if strings.ToLower(l) == "n" {
		return false
	}

	return d
}

func getArg(cmd *cobra.Command, args []string, index int) string {
	if len(args) <= index || args[index] == "" {
		_ = cmd.Usage()
		os.Exit(1)
	}

	return args[index]
}
