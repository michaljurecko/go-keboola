package strhelper

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jpillora/longestcommon"
	"github.com/umisama/go-regexpcache"
)

// FormatPathChange - example result "branch/config/{row -> row1}".
func FormatPathChange(src string, dst string, quote bool) string {
	q := ``
	if quote {
		q = `"`
	}

	prefix, src, dst := ExtractCommonPrefix(src, dst)
	if prefix != "" && len(prefix) > 2 {
		return fmt.Sprintf(`%s%s{%s -> %s}%s`, q, prefix, src, dst, q)
	} else {
		return fmt.Sprintf(`%s%s%s -> %s%s%s`, q, src, q, q, dst, q)
	}
}

// ExtractCommonPrefix from two strings.
// Returns prefix, first string without prefix and second string without prefix.
func ExtractCommonPrefix(a string, b string) (prefix string, ax string, bx string) {
	prefix = longestcommon.Prefix([]string{a, b})

	// Remove from the prefix everything after the last separator
	seps := regexp.QuoteMeta(`\/`)
	prefix = regexpcache.
		MustCompile(fmt.Sprintf(`(^|[%s])[^%s]*$`, seps, seps)).
		ReplaceAllString(prefix, "$1")

	// Prefix length > 2 AND must leave some different non-empty leftovers
	ax = strings.TrimPrefix(a, prefix)
	bx = strings.TrimPrefix(b, prefix)
	if len(prefix) > 2 && len(ax) > 0 && len(bx) > 0 && ax != bx {
		return prefix, ax, bx
	}

	return "", a, b
}
