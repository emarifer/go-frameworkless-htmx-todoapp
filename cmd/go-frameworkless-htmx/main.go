package main

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/emarifer/go-frameworkless-htmx/internal/db"
	"github.com/emarifer/go-frameworkless-htmx/internal/handlers"
	"github.com/emarifer/go-frameworkless-htmx/internal/services"
	"github.com/emarifer/go-frameworkless-htmx/internal/utils/prettylog"
)

func main() {
	logger := slog.New(prettylog.NewHandler(nil))

	mux := http.NewServeMux()

	// Setting the static file service (assets)
	fs := http.FileServer(http.Dir("./assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	// Dependency injection
	us := services.NewUserService(services.User{}, db.GetDB(logger))
	ah := handlers.NewAuthHandle(us)

	ts := services.NewTodoService(services.Todo{}, db.GetDB(logger))
	th := handlers.NewTodoHandle(ts)

	handlers.SetupRoutes(mux, logger, ah, th)

	logger.Info("🚀 Listening on :3000…")

	var s http.Handler = mux

	wrapped := handlers.LatencyLoggingMiddleware(s)

	log.Fatal(http.ListenAndServe(":3000", wrapped))
}

/* REFERENCES:
Logical operators in Go templates:
https://www.veriphor.com/articles/logical-operators/

Wildcard route patterns:
https://lets-go.alexedwards.net/sample/02.04-wildcard-route-patterns.html

What's the proper file extension or abbr. for golang's text/template?
https://stackoverflow.com/questions/22254013/whats-the-proper-file-extension-or-abbr-for-golangs-text-template

https://www.google.com/search?q=golang+chain+middleware&oq=golang+ch&aqs=chrome.1.69i57j35i19i39i512i650l2j69i65j69i60j69i65l2j69i60.7916j0j7&sourceid=chrome&ie=UTF-8
https://gist.github.com/husobee/fd23681261a39699ee37

https://www.google.com/search?q=golang+create+middleware&oq=golang+cr&aqs=chrome.2.69i57j69i59j35i19i39i512i650j69i65j69i61j69i60j69i65l2.9954j0j7&sourceid=chrome&ie=UTF-8
https://www.alexedwards.net/blog/making-and-using-middleware

https://www.google.com/search?q=golang+http+api+centralized+error&oq=golang+http+ap&aqs=chrome.1.69i57j35i39j69i60l3j69i65l3.10470j0j7&sourceid=chrome&ie=UTF-8
https://medium.com/@ozdemir.zynl/rest-api-error-handling-in-go-behavioral-type-assertion-509d93636afd

16-07-2024:
https://freshman.tech/snippets/go/extract-url-query-params/
https://groups.google.com/g/confd-users/c/0HfU_AYvGCY?pli=1
https://pkg.go.dev/text/template#hdr-Examples

19-07-2024:
https://www.google.com/search?q=golang+servemux+middleware&oq=&aqs=chrome.3.35i39i362l8.62006j0j7&sourceid=chrome&ie=UTF-8

https://drstearns.github.io/tutorials/gomiddleware/
https://www.mohitkhare.com/blog/go-middleware/
https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81
https://medium.com/@volodymyr.ladnik/adding-middleware-support-for-servemux-in-golang-fcc5f3901a26
https://www.jvt.me/posts/2023/09/01/golang-nethttp-global-middleware/
https://refactoring.guru/design-patterns/adapter/go/example
*/

// func serveTemplate(w http.ResponseWriter, r *http.Request) {
// 	lp := filepath.Join("views", "layout.html")
// 	fp := filepath.Join("views", filepath.Clean(r.URL.Path))

// 	data := map[string]any{
// 		"title": "Todo List",
// 		"name":  "Enrique Marín",
// 	}

// 	tmpl, _ := template.ParseFiles(lp, fp)
// 	tmpl.ExecuteTemplate(w, "layout", data)
// }
