package upper

import "unicode"

// Cap capitalizes the first character of the passed string if it is a letter.
func Cap(s string) string {
	r := []rune(s)
	if unicode.IsLetter(r[0]) {
		r[0] = unicode.ToUpper(r[0])
	}

	return string(r)
}
