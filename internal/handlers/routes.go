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

// buildChain builds the middlware chain recursively, functions are first class.
// This function allows us, therefore, to add as many middlewares
// as we want (included in a slice) that wrap our handler
// without the code becoming cumbersome to read.
func buildChain(f http.Handler, m ...middleware) http.Handler {
	// if our chain is done, use the original Handler type function
	if len(m) == 0 {
		return f
	}
	// otherwise nest the Handler type function
	return m[0](buildChain(f, m[1:cap(m)]...))
}

// renderView displays the log information corresponding
// to the controller and executes the template passed to it
func renderView(
	w http.ResponseWriter,
	r *http.Request,
	caller string,
	logger *slog.Logger,
	template string,
	data map[string]any,
) error {

	logger.Info(
		"🔵 Handler Info",
		"host", r.Host,
		"handler", caller,
		"method", r.Method,
		"path", r.URL.Path,
		"status", http.StatusOK,
	)

	return tmpl.ExecuteTemplate(w, template, data)
}

type apiError struct {
	message string
	status  int
	handler string
	logger  *slog.Logger
}

func (e apiError) Error() string {
	return e.message
}

// Use as a wrapper around the handler functions.
type rootHandle func(http.ResponseWriter, *http.Request) error

// rootHandle implements http.Handler interface.
// Since handlers return errors, the ServeHTTP implementation
// handles errors based on their type.
// Therefore, this function also performs centralized error handling.
func (fn rootHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	err := fn(w, r) // Call handler function
	if err == nil {
		return
	}

	if e, ok := err.(apiError); ok {
		e.logger.Error(
			"🔴 Handler Error",
			"host", r.Host,
			"handler", e.handler,
			"method", r.Method,
			"path", r.URL.Path,
			"status", e.status,
			"error", e.message,
		)

		// We handle the case that occurs when the user logs
		// in but cannot connect to the `todos` table of the DB.
		if strings.Contains(e.handler, "todoListHandle") {
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
	m *http.ServeMux, nfh *NotFoundHandle, ah *AuthHandle, th *TodoHandle,
) {
	if tmpl == nil {
		tmpl = template.Must(tmpl.ParseGlob("views/*.tmpl"))
	}

	// unprotected pages middlewares
	fp := []middleware{flagMiddleware}
	// protected pages middlewares
	p := []middleware{authMiddleware}

	// "/{$}" only matches the slash
	m.Handle("GET /{$}", buildChain(rootHandle(ah.homeHandle), fp...))

	m.Handle("GET /register", buildChain(rootHandle(ah.registerHandle), fp...))
	m.Handle("POST /register", buildChain(rootHandle(ah.registerPostHandle)))

	m.Handle("GET /login", buildChain(rootHandle(ah.loginHandle), fp...))
	m.Handle("POST /login", buildChain(rootHandle(ah.loginPostHandle)))
	m.Handle("POST /logout", buildChain(rootHandle(ah.logoutHandle), p...))

	m.Handle("GET /todo", buildChain(rootHandle(th.todoListHandle), p...))
	m.Handle("GET /create", buildChain(rootHandle(th.createTodoHandle), p...))
	m.Handle("POST /create", buildChain(
		rootHandle(th.createTodoPostHandle), p...,
	))
	m.Handle("GET /edit", buildChain(rootHandle(th.editTodoHandle), p...))
	m.Handle("POST /edit", buildChain(rootHandle(th.editTodoPostHandle), p...))
	m.Handle("DELETE /delete", buildChain(
		rootHandle(th.deleteTodoHandle), p...,
	))

	// "/" matches anything
	m.Handle("/", buildChain(rootHandle(nfh.notFoundHandle), fp...))
}
