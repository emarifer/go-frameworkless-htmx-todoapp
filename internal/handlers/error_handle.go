package handlers

import (
	"net/http"
)

func notFoundHandle(
	w http.ResponseWriter, r *http.Request,
) (string, error) {
	return asCaller(), apiError{
		status:  http.StatusNotFound,
		message: "error 404: not found",
	}
}
