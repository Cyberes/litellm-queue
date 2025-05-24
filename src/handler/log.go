package handler

import (
	"net/http"
)

func logRequest(req *http.Request) {
	// TODO: log model
	log.Infof("%s -- %s -- %s", req.RemoteAddr, req.Method, req.URL.Path)
}

func logAndReturnError(w http.ResponseWriter, r *http.Request, httpResponseStr string, code int, consoleStr ...string) {
	// consoleStr is optional.
	if len(consoleStr) > 0 {
		log.Infof("%s -- %s -- %s -- %s", r.RemoteAddr, r.Method, r.URL.Path, consoleStr[0])
	} else {
		log.Infof("%s -- %s -- %s -- %s", r.RemoteAddr, r.Method, r.URL.Path, httpResponseStr)
	}
	http.Error(w, httpResponseStr, code)
}
