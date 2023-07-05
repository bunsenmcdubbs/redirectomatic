package redirect

import (
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func Handler(s *Store) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/admin/", adminHandler(s))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		dest, err := s.Get(strings.TrimPrefix(r.URL.Path, "/"))
		if errors.Is(err, ErrNotFound) {
			// TODO configurable 404 page
			http.NotFound(w, r)
			return
		} else if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, dest.URL, http.StatusMovedPermanently)
	})

	return mux
}

//go:embed static
var embeddedFS embed.FS

func adminHandler(s *Store) http.Handler {
	// TODO implement authentication
	mux := http.NewServeMux()

	staticFS, _ := fs.Sub(embeddedFS, "static")
	mux.Handle("/admin/", http.FileServer(http.FS(staticFS)))

	mux.Handle("/admin/redirect", listRedirectHandler(s))

	mux.Handle("/admin/redirect/", http.StripPrefix("/admin/redirect/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			dest, err := s.Get(r.URL.Path)
			if errors.Is(err, ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, dest)
		} else if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			var dest RedirectDestination
			if err := json.Unmarshal(body, &dest); err != nil {
				log.Println("unable to parse input", r.URL.Path, err)
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			if _, err := url.ParseRequestURI(dest.URL); err != nil {
				log.Println("invalid destination url", r.URL.Path, err)
				http.Error(w, "invalid destination url", http.StatusBadRequest)
				return
			}
			if err := s.Upsert(r.URL.Path, dest); err != nil {
				log.Println("unable to update redirect", r.URL.Path, dest, err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		} else if r.Method == http.MethodDelete {
			s.Delete(r.URL.Path)
			w.WriteHeader(http.StatusOK)
		} else {
			http.NotFound(w, r)
			return
		}
	})))
	return mux
}

func listRedirectHandler(s *Store) http.Handler {
	// TODO paginate
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirects, err := s.List("", 0)
		if err != nil {
			log.Println("unable to list redirects", err)
			http.Error(w, "unable to list redirects", http.StatusInternalServerError)
			return
		}
		writeJSON(w, redirects)
	})
}

func writeJSON(w http.ResponseWriter, data any) {
	raw, _ := json.Marshal(data)
	_, _ = w.Write(raw)
	w.Header().Set("Content-Type", "application/json")
}
