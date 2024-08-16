package util

import "strings"

func SingleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// create a function that extracts the index name from the URL (example: /indexes/xxx/search)
func ExtractIndexName(url string) string {
	// split the URL by '/'
	split := strings.Split(url, "/")
	//
	if len(split) < 3 {
		return ""
	}
	// return the index name
	return split[2]
}
