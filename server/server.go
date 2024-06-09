package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Hukyl/genesis-kma-school-entry/server/config"
)

func NewServer(config config.Config, handler http.Handler) *http.Server {
	defaultTimeout := 15 * time.Second
	port := config.Port
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		WriteTimeout: defaultTimeout,
		ReadTimeout:  defaultTimeout,
	}
	slog.Info(fmt.Sprintf("Starting server on %s\n", server.Addr))
	return server
}
