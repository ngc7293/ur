package ur

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/exp/rand"
)

type Server struct {
	db Database
}

func GetSlugFromPath(path string) string {
	tokens := strings.Split(path, "/")
	return tokens[len(tokens)-1]
}

func GenerateSlug() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, 6)

	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}

	return string(b)
}

func ParsePostBody(r *http.Request) (string, error) {
	body, err := io.ReadAll(r.Body)

	if err != nil {
		return "", err
	}

	target, err := url.Parse(string(body))

	if err != nil {
		return "", err
	}

	return target.String(), nil
}

func NewServer(db Database) *Server {
	return &Server{db}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handler func(http.ResponseWriter, *http.Request) (int, error)

	switch r.URL.Path {
	case "/internal/metrics":
		switch r.Method {
		case "GET":
			handler = s.GetMetrics
		default:
			handler = s.MethodNotAllowed
		}
	default:
		switch r.Method {
		case "GET":
			handler = s.GetRoot
		case "POST":
			handler = s.PostRoot
		default:
			handler = s.NotFound
		}
	}

	status, err := handler(w, r)

	if err != nil {
		fmt.Printf(" error: %s\n", err)
	}

	fmt.Printf("access: %6s %s %d\n", r.Method, r.URL.Path, status)
}

func (s *Server) NotFound(w http.ResponseWriter, r *http.Request) (int, error) {
	w.WriteHeader(http.StatusNotFound)
	return http.StatusNotFound, nil
}

func (s *Server) MethodNotAllowed(w http.ResponseWriter, r *http.Request) (int, error) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	return http.StatusMethodNotAllowed, nil
}

func (s *Server) GetRoot(w http.ResponseWriter, r *http.Request) (int, error) {
	slug := GetSlugFromPath(r.URL.Path)
	url, err := s.db.GetUrlFromSlug(slug)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return http.StatusInternalServerError, err
	}

	if url == nil {
		w.WriteHeader(http.StatusNotFound)
		return http.StatusNotFound, nil
	}

	w.Header().Add("Location", url.Target)
	w.WriteHeader(http.StatusPermanentRedirect)
	err = s.db.IncrementUrlHitsFromSlug(slug)
	return http.StatusPermanentRedirect, err
}

func (s *Server) PostRoot(response http.ResponseWriter, request *http.Request) (int, error) {
	slug := GetSlugFromPath(request.URL.Path)

	if len(slug) == 0 {
		slug = GenerateSlug()
	}

	target, err := ParsePostBody(request)

	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return http.StatusInternalServerError, err
	}

	err = s.db.Insert(slug, target)

	if err != nil {
		var conflict *SlugConflictError
		if errors.As(err, &conflict) {
			response.WriteHeader(http.StatusConflict)
			return http.StatusConflict, nil
		}

		response.WriteHeader(http.StatusInternalServerError)
		return http.StatusInternalServerError, err
	}

	response.WriteHeader(http.StatusCreated)
	response.Write([]byte(slug))
	return http.StatusCreated, nil
}

func (s *Server) GetMetrics(w http.ResponseWriter, r *http.Request) (int, error) {
	urls, err := s.db.ListUrlWithHits()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return http.StatusInternalServerError, err
	}

	w.WriteHeader(http.StatusOK)

	for _, url := range urls {
		w.Write([]byte(fmt.Sprintf("url_hits{slug=\"%s\",target=\"%s\"} %d\n", url.Slug, url.Target, url.Hits)))
	}

	return http.StatusOK, nil
}

func Serve() {
	db, err := NewProgresDatabase(os.Getenv("DATABASE_URI"))

	if err != nil {
		panic("Failed to get connection pool")
	}

	server := NewServer(db)
	http.ListenAndServe(":8000", server)
}
