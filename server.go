package goweb

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/justinas/alice"
)

type Config struct {
	Router             *Router
	Port               string
	StaticFilesDirPath string
	ViewsDirPath       string
}

type ControllerFunc func(w http.ResponseWriter, r *http.Request)

type Router struct {
	routerMap map[string]ControllerFunc
}

func NewRouter() *Router {
	r := new(Router)
	r.routerMap = make(map[string]ControllerFunc)
	return r
}

func (r *Router) routes() map[string]ControllerFunc {
	return r.routerMap
}

func (r *Router) GET(path string, controller ControllerFunc) {
	r.routerMap[path] = controller
}

func Start(cfg Config) {
	log.Print("Setting up static file server")

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		ReadTimeout:  4 * time.Minute,
		WriteTimeout: 4 * time.Minute,
		Handler:      handler(cfg),
	}

	println("Server running...")
	if err := srv.ListenAndServe(); err != nil {
		log.Panicf("Error starting server: %s\n", err)
	}
}

func routes(cfg Config) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/css/", http.FileServer(http.Dir(cfg.StaticFilesDirPath)))

	for path, handler := range cfg.Router.routes() {
		mux.HandleFunc(path, handler)
	}

	return mux
}

func handler(cfg Config) http.Handler {
	handlers := []alice.Constructor{
		TimeoutHandler,
		RecoverHandler,
		RequestMetricsHandler,
		GZipHandler,
	}

	return alice.New(handlers...).Then(routes(cfg))
}

// RecoverHandler is a deferred function that will recover from the panic,
// respond with a HTTP 500 error and log the panic. When our code panics in production
// (make sure it should not but we can forget things sometimes) our application
// will shutdown. We must catch panics, log them and keep the application running.
// It's pretty easy with Go and our middleware system.
func RecoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rr := recover(); rr != nil {
				var err error
				switch x := rr.(type) {
				case string:
					err = errors.New(x)
				case error:
					err = x
				default:
					err = errors.New("unknown panic")
				}
				eh := ErrorHandler{}
				if err != nil {
					perr := fmt.Errorf("PANIC: %s", err.Error())
					eh.HandleError(r, perr)

					// send stack trace as well
					if etrace := debug.Stack(); etrace != nil {
						etrace := fmt.Errorf("STACKTRACE: %s", debug.Stack())
						debug.PrintStack()
						eh.HandleError(r, etrace)
					}
				}
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		if next != nil {
			next.ServeHTTP(w, r)
		}
	}

	return http.HandlerFunc(fn)
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GZipHandler(h http.Handler) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r) // serve the original request
			return
		}

		w.Header().Set("Content-Encoding", "gzip")

		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		h.ServeHTTP(gzw, r) // serve the original request
	}
	return http.HandlerFunc(f)
}

func TimeoutHandler(h http.Handler) http.Handler {
	return http.TimeoutHandler(h, 4*time.Second, "timed out")
}

func RequestMetricsHandler(h http.Handler) http.Handler {
	logFn := func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()

		uri := r.RequestURI
		method := r.Method

		h.ServeHTTP(rw, r) // serve the original request

		duration := time.Since(start)

		// log request details
		log.Printf("Request: %s %s %d", uri, method, duration)
	}

	return http.HandlerFunc(logFn)
}

// ErrorHandler Error handler for routers and middlewares
type ErrorHandler struct {
	PanicHandler bool
}

// HandleError handles the error or panic on the middleware
func (eh ErrorHandler) HandleError(r *http.Request, err error) {
	if eh.PanicHandler {
		msg := fmt.Sprintf("URI: %s, %s", r.RequestURI, err)
		log.Printf("Panic: %s", msg)
	} else {
		agent := "Unknown"
		headers := r.Header["User-Agent"]
		if len(headers) > 0 {
			agent = headers[0]
		}

		msg := fmt.Sprintf("Server Error - Method: %s, User Agent: %s, Remote Address: %s, Endpoint: %s",
			r.Method,
			agent,
			r.RemoteAddr,
			r.RequestURI)

		if err != nil {
			msg = fmt.Sprintf("%s, ERROR: %s", msg, err)
			log.Printf("Panic: %s", msg)
		}
	}
}
