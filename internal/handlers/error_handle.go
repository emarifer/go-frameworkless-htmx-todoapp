package handlers

import (
	"log/slog"
	"net/http"
)

func NewNotFoundHandle(l *slog.Logger) *NotFoundHandle {

	return &NotFoundHandle{
		logger: l,
	}
}

type NotFoundHandle struct {
	logger *slog.Logger
}

func (nfh *NotFoundHandle) notFoundHandle(
	w http.ResponseWriter, r *http.Request,
) error {

	return apiError{
		message: "error 404: not found",
		status:  http.StatusNotFound,
		handler: asCaller(),
		logger:  nfh.logger,
	}
}
