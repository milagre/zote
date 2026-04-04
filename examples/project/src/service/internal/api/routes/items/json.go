package items

import "net/http"

func jsonHeader() http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json; charset=utf-8")
	return h
}
