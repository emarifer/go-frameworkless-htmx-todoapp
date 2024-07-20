package handlers

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"
)

var tmpl *template.Template

// middleware is a definition of what a middleware is,
// take in one type Handler interface and
// wrap it within another Handler interface
type middleware func(http.Handler) http.Handler

// chainingMiddleware chains the middlewares that
// we pass to it in a slice wrapping the handle
// that we pass as the first parameter.
// This function allows us, therefore, to add as many middlewares
// as we want (included in a slice) that wrap our handler
// without the code becoming cumbersome to read.
func chainingMiddleware(h http.Handler, m ...middleware) http.Handler {
	if len(m) < 1 {
		return h
	}

	wrappedHandler := h
	for i := len(m) - 1; i >= 0; i-- {
		wrappedHandler = m[i](wrappedHandler)
	}

	return wrappedHandler
}

type apiError struct {
	status  int
	message string
}

func (e apiError) Error() string {
	return e.message
}

// Use as a wrapper around the handler functions.
// Basically, they use the Adapter design pattern.
type adapterHandle struct {
	l *slog.Logger
	h func(http.ResponseWriter, *http.Request) (string, error)
}

func newAdapterHandle(
	l *slog.Logger,
	h func(http.ResponseWriter, *http.Request) (string, error),
) *adapterHandle {
	return &adapterHandle{l, h}
}

// adapterHandle implements the http.Handler interface.
// Because handlers return errors, the ServeHTTP implementation
// handles errors based on their type. Therefore, this function
// also performs centralized error handling in addition
// to successfully logging the handlers' output.
func (a adapterHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	caller, err := a.h(w, r)
	if err == nil {
		a.l.Info(
			"🔵 Handler Info",
			"host", r.Host,
			"user_agent", r.Header.Get("User-Agent"),
			"handler", caller,
			"method", r.Method,
			"path", r.URL.Path,
			"status", http.StatusOK,
		)
		return
	}

	if e, ok := err.(apiError); ok {
		a.l.Error(
			"🔴 Handler Error",
			"host", r.Host,
			"user_agent", r.Header.Get("User-Agent"),
			"handler", caller,
			"method", r.Method,
			"path", r.URL.Path,
			"status", e.status,
			"error", e.message,
		)

		// We handle the case that occurs when the user logs
		// in but cannot connect to the `todos` table of the DB.
		if strings.Contains(caller, "todoListHandle") {
			dc := &http.Cookie{
				Name:    "jwt",
				Path:    "/",
				MaxAge:  -1,
				Expires: time.Unix(1, 0),
			}

			http.SetCookie(w, dc)
		}

		data := map[string]any{
			"isError":       true,
			"fromProtected": requestFromProtected(r.Context()),
		}

		switch e.status {
		case 404:
			w.WriteHeader(http.StatusNotFound)
			data["title"] = "| Error 404"
			tmpl.ExecuteTemplate(w, "error_404.tmpl", data)
			return
		case 500:
			w.WriteHeader(http.StatusInternalServerError)
			data["title"] = "| Error 500"
			tmpl.ExecuteTemplate(w, "error_500.tmpl", data)
			return
		}
	}

	// In case the error returned by the handler is not of
	// type apiError (it has to be an error while
	// rendering the template), we return a JSON response with a code 500.
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "failure",
		"message": "Unknown server error",
		"code":    http.StatusInternalServerError,
	})
}

// asCaller gets the caller from which this function is called.
// The index `1` returns the name of the handler we are looking for.
func asCaller() string {
	pc, _, _, _ := runtime.Caller(1) // index = 1
	hs := strings.Split(runtime.FuncForPC(pc).Name(), "/")

	return hs[len(hs)-1]
}

// SetupRoutes starts the `tmpl` variable,
// necessary to execute the various templates that
// the handlers will execute, while registering
// the routes of the various endpoints.
func SetupRoutes(
	m *http.ServeMux,
	l *slog.Logger,
	ah *AuthHandle,
	th *TodoHandle,
) {
	if tmpl == nil {
		tmpl = template.Must(tmpl.ParseGlob("views/*.tmpl"))
	}

	// unprotected pages middlewares
	up := []middleware{flagMiddleware}
	// protected pages middlewares
	p := []middleware{flagMiddleware, authMiddleware}

	ad := newAdapterHandle(l, ah.homeHandle)
	// "/{$}" only matches the slash
	m.Handle("GET /{$}", chainingMiddleware(ad, up...))

	ad = newAdapterHandle(l, ah.registerHandle)
	m.Handle("GET /register", chainingMiddleware(ad, up...))
	ad = newAdapterHandle(l, ah.registerPostHandle)
	m.Handle("POST /register", chainingMiddleware(ad))

	ad = newAdapterHandle(l, ah.loginHandle)
	m.Handle("GET /login", chainingMiddleware(ad, up...))
	ad = newAdapterHandle(l, ah.loginPostHandle)
	m.Handle("POST /login", chainingMiddleware(ad))
	ad = newAdapterHandle(l, ah.logoutHandle)
	m.Handle("POST /logout", chainingMiddleware(ad, p...))

	ad = newAdapterHandle(l, th.todoListHandle)
	m.Handle("GET /todo", chainingMiddleware(ad, p...))
	ad = newAdapterHandle(l, th.createTodoHandle)
	m.Handle("GET /create", chainingMiddleware(ad, p...))
	ad = newAdapterHandle(l, th.createTodoPostHandle)
	m.Handle("POST /create", chainingMiddleware(ad, p...))
	ad = newAdapterHandle(l, th.editTodoHandle)
	m.Handle("GET /edit", chainingMiddleware(ad, p...))
	ad = newAdapterHandle(l, th.editTodoPostHandle)
	m.Handle("POST /edit", chainingMiddleware(ad, p...))
	ad = newAdapterHandle(l, th.deleteTodoHandle)
	m.Handle("DELETE /delete", chainingMiddleware(ad, p...))

	ad = newAdapterHandle(l, notFoundHandle)
	// "/" matches anything
	m.Handle("/", chainingMiddleware(ad, up...))
}
