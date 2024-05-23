package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"text/template"

	"github.com/a-h/templ"
	"github.com/acaloiaro/hugo-htmx-go-template/partials"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed all:public
var content embed.FS

func main() {
	// mux := http.NewServeMux()
	serverRoot, _ := fs.Sub(content, "public")

	app := pocketbase.New()

	// serves static files from the provided public dir (if exists)
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.Use(middleware.CORS())
		e.Router.GET("/*", apis.StaticDirectoryHandler(serverRoot, false))
		e.Router.Any("/goodbyeworld.html", GoodbyeWorldHandler)
		e.Router.Any("/hello_world", echo.WrapHandler(HelloWorld{}))
		e.Router.Any("/hello_world_form", echo.WrapHandler(HelloWorldForm{}))
		return nil
	})

	app.Bootstrap()
	if _, err := apis.Serve(app, apis.ServeConfig{
		HttpAddr:        "127.0.0.1:1314",
		ShowStartBanner: true,
	}); err != nil {
		log.Fatal(err)
	}

	// Serve all hugo content (the 'public' directory) at the root url
	// mux.Handle("/", http.FileServer(http.FS(serverRoot)))

	// cors := func(h http.Handler) http.HandlerFunc {
	// 	return func(w http.ResponseWriter, r *http.Request) {
	// 		// in development, the Origin is the the Hugo server, i.e. http://localhost:1313
	// 		// but in production, it is the domain name where one's site is deployed
	// 		//
	// 		// CHANGE THIS: You likely do not want to allow any origin (*) in production. The value should be the base URL of
	// 		// where your static content is served
	// 		w.Header().Set("Access-Control-Allow-Origin", "*")
	// 		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
	// 		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, hx-target, hx-current-url, hx-request")
	// 		if r.Method == "OPTIONS" {
	// 			w.WriteHeader(http.StatusNoContent)
	// 			return
	// 		}
	// 		h.ServeHTTP(w, r)
	// 	}
	// }

	// Add any number of handlers for custom endpoints here
	// mux.HandleFunc("/goodbyeworld.html", cors(templ.Handler(partials.GoodbyeWorld())))
	// mux.HandleFunc("/hello_world", cors(http.HandlerFunc(helloWorld)))
	// mux.HandleFunc("/hello_world_form", cors(http.HandlerFunc(helloWorldForm)))

	// fmt.Printf("Starting API server on port 1314\n")
	// if err := http.ListenAndServe("0.0.0.0:1314", mux); err != nil {
	// 	log.Fatal(err)
	// }
}

// This custom Render replaces Echo's echo.Context.Render() with templ's templ.Component.Render().
func Render(ctx echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}

func GoodbyeWorldHandler(c echo.Context) error {
	return Render(c, http.StatusOK, partials.GoodbyeWorld())
}

type HelloWorld struct{}

// the handler accepts GET requests to /hello_world
// It checks the URL params for the "name" param and populates the html/template variable with its value
// if no "name" url parameter is present, "name" is defaulted to "World"
//
// It responds with the the HTML partial `partials/helloworld.html`
func (HelloWorld) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "null" || name == "" {
		name = "World"
	}

	tmpl := template.Must(template.ParseFiles("partials/helloworld.html"))
	buff := bytes.NewBufferString("")
	err := tmpl.Execute(buff, map[string]string{"Name": name})
	if err != nil {
		ise(err, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(buff.Bytes())
}

type HelloWorldForm struct{}

// this handler accepts POST requests to /hello_world_form
// It checks the post request body for the form value "name" and populates the html/template
// variable with its value
//
// It responds with a simple greeting HTML partial
func (HelloWorldForm) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := "World"
	if err := r.ParseForm(); err != nil {
		ise(err, w)
		return
	}

	name = r.FormValue("name")
	if err := partials.HelloWorldGreeting(name).Render(r.Context(), w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func ise(err error, w http.ResponseWriter) {
	fmt.Fprintf(w, "error: %v", err)
	w.WriteHeader(http.StatusInternalServerError)
}
