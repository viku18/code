package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/acai-travel/tech-challenge/internal/chat"
	"github.com/acai-travel/tech-challenge/internal/chat/assistant"
	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/acai-travel/tech-challenge/internal/httpx"
	"github.com/acai-travel/tech-challenge/internal/mongox"
	"github.com/acai-travel/tech-challenge/internal/pb"
	"github.com/acai-travel/tech-challenge/internal/telemetry"
	"github.com/gorilla/mux"
	"github.com/twitchtv/twirp"
)

func main() {
	// Initialize metrics
	telemetry.Init()
	slog.Info("OpenTelemetry metrics initialized")

	mongo := mongox.MustConnect()

	repo := model.New(mongo)
	assist := assistant.New()

	server := chat.NewServer(repo, assist)

	// Configure handler
	handler := mux.NewRouter()
	handler.Use(
		httpx.Logger(),
		httpx.Recovery(),
	)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hi, my name is Server!")
	})

	// Add metrics endpoint
	handler.Handle("/metrics", telemetry.GetMetricsHandler())

	handler.PathPrefix("/twirp/").Handler(pb.NewChatServiceServer(server, twirp.WithServerJSONSkipDefaults(true)))

	// Start the server
	slog.Info("Starting the server on :8080...")
	slog.Info("Metrics available at: http://localhost:8080/metrics")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}