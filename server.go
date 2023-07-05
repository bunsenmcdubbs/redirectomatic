package redirect

import (
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
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

//go:embed frontend
var embeddedFS embed.FS

func adminHandler(s *Store) http.Handler {
	// TODO implement authentication
	mux := http.NewServeMux()

	mux.Handle("/admin/", http.StripPrefix("/admin", uiHandler(s)))

	staticFS, _ := fs.Sub(embeddedFS, "frontend/static")
	mux.Handle("/admin/static/", http.StripPrefix("/admin/static/", http.FileServer(http.FS(staticFS))))

	mux.Handle("/admin/api/", http.StripPrefix("/admin/api", apiHandler(s)))

	return mux
}

func uiHandler(s *Store) http.Handler {
	mux := http.NewServeMux()

	t := template.Must(template.ParseFS(embeddedFS, "frontend/templates/index.html"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			key := r.Form.Get("key")
			dest := r.Form.Get("url")
			if key == "" || dest == "" {
				http.Error(w, "invalid input: both key and url must be present", http.StatusBadRequest)
				return
			}
			if _, err := url.ParseRequestURI(dest); err != nil {
				http.Error(w, "invalid destination url", http.StatusBadRequest)
				return
			}

			if err := s.Upsert(key, RedirectDestination{
				URL: dest,
			}); err != nil {
				http.Error(w, "unable to save redirect", http.StatusInternalServerError)
				return
			}
		}

		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		redirects, _ := s.List("", 0)
		_ = t.Execute(w, struct {
			Redirects []Redirect
		}{Redirects: redirects})
	})

	return mux
}

func apiHandler(s *Store) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/redirects", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// TODO paginate
		redirects, err := s.List("", 0)
		if err != nil {
			log.Println("unable to list redirects", err)
			http.Error(w, "unable to list redirects", http.StatusInternalServerError)
			return
		}
		writeJSON(w, redirects)
	}))

	mux.Handle("/redirects/", http.StripPrefix("/redirects/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			if _, err := url.ParseRequestURI(dest.URL); err != nil {
				http.Error(w, "invalid destination url", http.StatusBadRequest)
				return
			}
			dest.UpdatedAt = time.Now()
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
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	})))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL)
		mux.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, data any) {
	raw, _ := json.Marshal(data)
	_, _ = w.Write(raw)
	w.Header().Set("Content-Type", "application/json")
}
