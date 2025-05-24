package handler

import (
	"net/http"
)

func logRequest(req *http.Request) {
	// TODO: log model
	log.Infof("%s -- %s -- %s", req.RemoteAddr, req.Method, req.URL.Path)
}

func logAndReturnError(w http.ResponseWriter, httpResponseStr string, code int, consoleStr ...string) {
	// consoleStr is optional.
	if len(consoleStr) > 0 {
		log.Errorln(consoleStr[0])
	} else {
		log.Errorln(httpResponseStr)
	}
	http.Error(w, httpResponseStr, code)
}
