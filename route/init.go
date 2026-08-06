package route

import (
	"net/http"
)

type Config struct{}

func Initialize(config Config) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /health", Health())

	return mux
}
