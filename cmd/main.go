package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"server-static/internal/config"
	"server-static/internal/db/database"
	"server-static/internal/handlers"
	"server-static/internal/logger"
	"server-static/internal/middleware"
	"server-static/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	log.Println("🚀 Starting Web Content Server")

	_ = godotenv.Load()
	config.Load()

	logCloser, err := logger.Init(config.AppConfig.LogPath)
	if err != nil {
		log.Printf("⚠️ File logging disabled: %v", err)
	} else {
		defer logCloser.Close()
		log.Printf("📝 Logging to: %s (max 25MB per file)", config.AppConfig.LogPath)
	}

	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer database.Disconnect()
	log.Println("✅ MongoDB connected")

	port := config.AppConfig.Port
	if port == "" {
		port = "8082"
	}

	go services.StartSettingSyncScheduler(context.Background())

	h := handlers.NewHandler(handlers.Handler{})

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"server-static"}`)
	})
	mux.HandleFunc("/logs", h.HandleLogList)
	mux.HandleFunc("/logs/", h.HandleLogFile)
	mux.HandleFunc("/playlist/", h.PlaylistJSON)
	mux.HandleFunc("/advert/", h.AdvertJSON)
	mux.HandleFunc("/vast/", h.Vast)
	mux.HandleFunc("/image/", h.ImageAdverts)
	mux.HandleFunc("/script/", h.ScriptAdverts)
	mux.HandleFunc("/", h.Home)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: middleware.CORS(mux),
	}

	fmt.Printf("Server started at http://localhost:%s\n", port)
	log.Printf("📍 Endpoints:")
	log.Printf("   GET /playlist/{fileSlug}.json  - JW Player playlist feed")
	log.Printf("   GET /advert/{adSlug}.json      - Unified advert feed")

	if err := server.ListenAndServe(); err != nil {
		log.Println("Error starting server:", err)
	}
}
