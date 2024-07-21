package handlers

import (
	"net/http"
)

func notFoundHandle(w http.ResponseWriter, r *http.Request) error {
	message := "error 404: not found"
	w.Header().Add(HEADER_KEY_HANDLER, asCaller())
	w.Header().Add(HEADER_KEY_ERRMSG, message)
	w.WriteHeader(http.StatusNotFound)
	return apiError{status: http.StatusNotFound, message: message}
}
