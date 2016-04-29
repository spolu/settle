package logging

import (
	"net/http"
	"sort"
	"strings"

	"golang.org/x/net/context"
)

// LogHeader properly logs headers with a prefix
func LogHeader(
	c context.Context,
	prefix string,
	values http.Header,
) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	buf := make([]string, 0, len(values))
	for _, k := range keys {
		buf = append(buf, stringifyKeys(k, values[k])...)
	}
	Log(c, prefix+strings.Join(buf, " "))
}
