package commands

import (
	"fmt"
	"regexp"
	"strings"
)

func Sprintf(f string, i ...interface{}) string {
	return fmt.Sprintf(f, i...)
}

var whitespaceReg = regexp.MustCompile("\\s+")

func SplitWhitespace(s string) []string {
	return whitespaceReg.Split(s, -1)
}

func Capitalize(s string) string {
	return strings.ToUpper(string([]rune(s)[0:1])) + s[1:]
}
