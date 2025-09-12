package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	
	"github.com/asmit27rai/kubesight/internal/api"
	"github.com/asmit27rai/kubesight/internal/config"
	"github.com/asmit27rai/kubesight/internal/engine"
	"github.com/asmit27rai/kubesight/internal/sampling"
	"github.com/asmit27rai/kubesight/internal/stream"
)

func main() {
	log.Println("🚀 Starting KubeSight Approximate Query Engine...")

	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	engineConfig := engine.QueryEngineConfig{
		HLLPrecision:   uint8(cfg.Storage.HLLPrecision),
		CMSWidth:       uint32(cfg.Storage.CMSWidth),
		CMSDepth:       uint32(cfg.Storage.CMSDepth),
		BloomSize:      uint32(cfg.Storage.BloomSize),
		BloomHashes:    uint32(cfg.Storage.BloomHashes),
		SamplingConfig: sampling.SamplingConfig{
			BaseRate:       cfg.Sampling.DefaultRate,
			AnomalyRate:    cfg.Sampling.IncidentRate,
			WindowSize:     time.Duration(cfg.Sampling.WindowSizeMin) * time.Minute,
			ReservoirSize:  cfg.Sampling.ReservoirSize,
		},
	}

	queryEngine := engine.NewQueryEngine(engineConfig)
	log.Printf("✅ Query Engine initialized with HLL precision: %d, CMS: %dx%d", 
		cfg.Storage.HLLPrecision, cfg.Storage.CMSWidth, cfg.Storage.CMSDepth)

	streamConfig := stream.ProcessorConfig{
		KafkaBrokers: cfg.Kafka.Brokers,
		Topics: stream.Topics{
			Metrics: cfg.Kafka.Topics.Metrics,
			Logs:    cfg.Kafka.Topics.Logs,
			Events:  cfg.Kafka.Topics.Events,
		},
		QueryEngine: queryEngine,
	}

	processor, err := stream.NewProcessor(streamConfig)
	if err != nil {
		log.Fatalf("Failed to create stream processor: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Starting stream processor...")
		if err := processor.Start(ctx); err != nil {
			log.Printf("Stream processor error: %v", err)
		}
	}()

	apiHandler := api.NewHandler(queryEngine)
	router := mux.NewRouter()

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	api.RegisterRoutes(apiRouter, apiHandler)

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))
	router.HandleFunc("/", serveDashboard)
	router.HandleFunc("/health", healthCheck)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("HTTP server starting on %s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Dashboard available at: http://%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Printf("API available at: http://%s:%d/api/v1", cfg.Server.Host, cfg.Server.Port)
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	printStartupSummary(cfg)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/dashboard.html")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "healthy", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
}

func printStartupSummary(cfg *config.Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("KubeSight Approximate Query Engine")
	log.Println(strings.Repeat("=", 60))
	log.Printf("Server: http://%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Kafka Brokers: %v", cfg.Kafka.Brokers)
	log.Printf("Default Sampling Rate: %.2f%%", cfg.Sampling.DefaultRate*100)
	log.Printf("Anomaly Sampling Rate: %.2f%%", cfg.Sampling.IncidentRate*100)
	log.Printf("HyperLogLog Precision: %d (±%.2f%% error)", 
		cfg.Storage.HLLPrecision, 
		1.04/math.Sqrt(math.Pow(2, float64(cfg.Storage.HLLPrecision)))*100)
	log.Printf("Count-Min Sketch: %dx%d", cfg.Storage.CMSWidth, cfg.Storage.CMSDepth)
	log.Printf("Bloom Filter: %d elements, %d hash functions", 
		cfg.Storage.BloomSize, cfg.Storage.BloomHashes)
	log.Println(strings.Repeat("=", 60))
	log.Println("Ready to process approximate queries!")
	log.Println("Try these sample queries:")
	log.Println("   • GET /api/v1/query?type=count_distinct&metric=pod_name")
	log.Println("   • GET /api/v1/query?type=percentile&metric=cpu_usage&p=95")
	log.Println("   • GET /api/v1/query?type=top_k&metric=memory_usage&k=10")
	log.Println(strings.Repeat("=", 60) + "\n")
}