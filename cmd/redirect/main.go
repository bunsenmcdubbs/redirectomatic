package main

import (
	"github.com/bunsenmcdubbs/redirectomatic"
	"log"
	"net/http"
	"time"
)

func main() {
	store, _ := redirect.OpenStore("./redirect.db")

	s := &http.Server{
		Addr:    ":8080",
		Handler: logger(redirect.Handler(store)),
	}
	log.Fatal(s.ListenAndServe())
}

type responseRecorder struct {
	http.ResponseWriter
	code int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func logger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := responseRecorder{ResponseWriter: w}
		handler.ServeHTTP(&recorder, r)
		elapsed := time.Now().Sub(start)
		log.Println(r.Method, r.URL, elapsed, recorder.code)
	})
}
