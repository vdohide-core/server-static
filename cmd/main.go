package main

import (
	"fmt"
	"log"
	"net/http"
	"server-static/internal/config"
	"server-static/internal/db/database"
	"server-static/internal/handlers"
	"server-static/internal/logger"
	"server-static/internal/services"
	"context"

	"github.com/joho/godotenv"
)

func main() {
	log.Println("🚀 Starting Web Content Server")

	// Load .env (optional)
	_ = godotenv.Load()

	// Load config
	config.Load()

	// Init file logger (writes to stdout + rotating log file)
	logCloser, err := logger.Init(config.AppConfig.LogPath)
	if err != nil {
		log.Printf("⚠️ File logging disabled: %v", err)
	} else {
		defer logCloser.Close()
		log.Printf("📝 Logging to: %s (max 25MB per file)", config.AppConfig.LogPath)
	}

	// Connect to MongoDB
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer database.Disconnect()
	log.Println("✅ MongoDB connected")

	// Get port from environment or use default
	port := config.AppConfig.Port
	if port == "" {
		port = "8082"
	}

	// Start Settings Sync Scheduler
	go services.StartSettingSyncScheduler(context.Background())

	// Initialize handlers
	h := handlers.NewHandler(handlers.Handler{})

	// Routes
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"server-static"}`)
	})
	http.HandleFunc("/logs", h.HandleLogList)
	http.HandleFunc("/logs/", h.HandleLogFile)
	http.HandleFunc("/vast/", h.Vast)
	http.HandleFunc("/image/", h.ImageAdverts)
	http.HandleFunc("/script/", h.ScriptAdverts)
	http.HandleFunc("/", h.Home)

	fmt.Printf("Server started at http://localhost:%s\n", port)

	// CORS middleware
	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Range")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.DefaultServeMux.ServeHTTP(w, r)
	})

	if err := http.ListenAndServe(":"+port, corsHandler); err != nil {
		log.Println("Error starting server:", err)
	}
}
