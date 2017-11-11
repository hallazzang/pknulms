package pknulms

import "strings"

func splitString(s, sep string) (string, string) {
	t := strings.SplitN(s, sep, 2)
	return t[0], t[1]
}
