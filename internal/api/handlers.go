package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	
	"github.com/asmit27rai/kubesight/internal/engine"
	"github.com/asmit27rai/kubesight/pkg/metrics"
)

type Handler struct {
	queryEngine *engine.QueryEngine
}

func NewHandler(queryEngine *engine.QueryEngine) *Handler {
	return &Handler{
		queryEngine: queryEngine,
	}
}

func RegisterRoutes(router *mux.Router, handler *Handler) {
	router.HandleFunc("/query", handler.ExecuteQuery).Methods("GET", "POST")
	router.HandleFunc("/query/batch", handler.ExecuteBatchQuery).Methods("POST")
	
	router.HandleFunc("/stats", handler.GetStats).Methods("GET")
	router.HandleFunc("/stats/engine", handler.GetEngineStats).Methods("GET")
	router.HandleFunc("/stats/sampling", handler.GetSamplingStats).Methods("GET")
	
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")
	router.HandleFunc("/metrics", handler.GetMetrics).Methods("GET")
	
	router.HandleFunc("/samples", handler.GetSamples).Methods("GET")
	router.HandleFunc("/samples/{stratum}", handler.GetStratumSamples).Methods("GET")
	
	router.HandleFunc("/demo/generate", handler.GenerateTestData).Methods("POST")
	router.HandleFunc("/demo/query", handler.DemoQuery).Methods("GET")
}

func (h *Handler) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	var request *metrics.QueryRequest
	
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			h.writeError(w, http.StatusBadRequest, "Invalid JSON request", err)
			return
		}
	} else {
		request = h.parseQueryParams(r)
		if request == nil {
			h.writeError(w, http.StatusBadRequest, "Missing required query parameters", nil)
			return
		}
	}

	if request.ID == "" {
		request.ID = fmt.Sprintf("query_%d", time.Now().UnixNano())
	}

	result, err := h.queryEngine.ExecuteQuery(request)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Query execution failed", err)
		return
	}

	h.writeJSON(w, http.StatusOK, result)
	
	log.Printf("Query executed: %s (type: %s, time: %v, samples: %d)", 
		request.ID, request.QueryType, result.ProcessingTime, result.SampleSize)
}

func (h *Handler) ExecuteBatchQuery(w http.ResponseWriter, r *http.Request) {
	var requests []metrics.QueryRequest
	
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON request", err)
		return
	}

	results := make([]*metrics.QueryResult, len(requests))
	
	for i, request := range requests {
		if request.ID == "" {
			request.ID = fmt.Sprintf("batch_query_%d_%d", time.Now().UnixNano(), i)
		}
		
		result, err := h.queryEngine.ExecuteQuery(&request)
		if err != nil {
			result = &metrics.QueryResult{
				ID:            request.ID,
				Query:         request.Query,
				Result:        nil,
				SampleSize:    0,
				IsApproximate: false,
				Timestamp:     time.Now(),
			}
		}
		results[i] = result
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.queryEngine.GetStats()
	
	systemStats := metrics.SystemStats{
		Timestamp:        time.Now(),
		TotalMetrics:     stats.TotalSamples,
		SampledMetrics:   stats.TotalSamples,
		SamplingRate:     0.05,
		ProcessingRate:   float64(stats.TotalSamples) / time.Since(stats.LastUpdateTime).Seconds(),
		QueryLatencyP95:  float64(stats.AvgLatency.Nanoseconds()) / 1e6,
		ErrorRate:        stats.ErrorRate,
	}

	h.writeJSON(w, http.StatusOK, systemStats)
}

func (h *Handler) GetEngineStats(w http.ResponseWriter, r *http.Request) {
	stats := h.queryEngine.GetStats()
	h.writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) GetSamplingStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"total_processed":  1000000,
		"total_sampled":    50000,
		"sampling_rate":    0.05,
		"adaptive_enabled": true,
		"reservoirs":       5,
	}
	
	h.writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().Format(time.RFC3339),
		"version":     "1.0.0",
		"service":     "kubesight-query-engine",
	}
	
	h.writeJSON(w, http.StatusOK, status)
}

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	stats := h.queryEngine.GetStats()
	
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	
	fmt.Fprintf(w, "# HELP kubesight_queries_total Total number of queries processed\n")
	fmt.Fprintf(w, "# TYPE kubesight_queries_total counter\n")
	fmt.Fprintf(w, "kubesight_queries_total{type=\"total\"} %d\n", stats.TotalQueries)
	fmt.Fprintf(w, "kubesight_queries_total{type=\"approximate\"} %d\n", stats.ApproxQueries)
	
	fmt.Fprintf(w, "# HELP kubesight_query_duration_milliseconds Query processing time\n")
	fmt.Fprintf(w, "# TYPE kubesight_query_duration_milliseconds histogram\n")
	fmt.Fprintf(w, "kubesight_query_duration_milliseconds_sum %f\n", float64(stats.AvgLatency.Nanoseconds())/1e6)
	fmt.Fprintf(w, "kubesight_query_duration_milliseconds_count %d\n", stats.TotalQueries)
	
	fmt.Fprintf(w, "# HELP kubesight_samples_total Total number of samples processed\n")
	fmt.Fprintf(w, "# TYPE kubesight_samples_total counter\n")
	fmt.Fprintf(w, "kubesight_samples_total %d\n", stats.TotalSamples)
}

func (h *Handler) GetSamples(w http.ResponseWriter, r *http.Request) {
	samples := map[string]interface{}{
		"total_samples": 1000,
		"strata_count":  5,
		"last_updated":  time.Now(),
	}
	
	h.writeJSON(w, http.StatusOK, samples)
}

func (h *Handler) GetStratumSamples(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stratum := vars["stratum"]
	
	if stratum == "" {
		h.writeError(w, http.StatusBadRequest, "Missing stratum parameter", nil)
		return
	}
	
	samples := map[string]interface{}{
		"stratum":       stratum,
		"sample_count":  100,
		"samples":       []interface{}{},
		"last_updated":  time.Now(),
	}
	
	h.writeJSON(w, http.StatusOK, samples)
}

func (h *Handler) GenerateTestData(w http.ResponseWriter, r *http.Request) {
	var config struct {
		Count     int    `json:"count"`
		ClusterID string `json:"cluster_id"`
		Namespace string `json:"namespace"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON request", err)
		return
	}
	
	if config.Count <= 0 {
		config.Count = 1000
	}
	if config.ClusterID == "" {
		config.ClusterID = "test-cluster"
	}
	if config.Namespace == "" {
		config.Namespace = "default"
	}
	
	go h.generateTestMetrics(config.Count, config.ClusterID, config.Namespace)
	
	response := map[string]interface{}{
		"message":     "Test data generation started",
		"count":       config.Count,
		"cluster_id":  config.ClusterID,
		"namespace":   config.Namespace,
		"status":      "generating",
	}
	
	h.writeJSON(w, http.StatusAccepted, response)
}

func (h *Handler) DemoQuery(w http.ResponseWriter, r *http.Request) {
	queryType := r.URL.Query().Get("type")
	if queryType == "" {
		queryType = "count_distinct"
	}
	
	var request *metrics.QueryRequest
	
	switch queryType {
	case "count_distinct":
		request = &metrics.QueryRequest{
			ID:        "demo_count_distinct",
			Query:     "COUNT_DISTINCT(pod_name)",
			QueryType: metrics.CountDistinct,
		}
	case "percentile":
		request = &metrics.QueryRequest{
			ID:        "demo_percentile",
			Query:     "PERCENTILE(95) cpu_usage",
			QueryType: metrics.Percentile,
		}
	case "top_k":
		request = &metrics.QueryRequest{
			ID:        "demo_top_k",
			Query:     "TOP_K(10) memory_usage",
			QueryType: metrics.TopK,
		}
	default:
		h.writeError(w, http.StatusBadRequest, "Unknown demo query type", nil)
		return
	}
	
	result, err := h.queryEngine.ExecuteQuery(request)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Demo query failed", err)
		return
	}
	
	h.writeJSON(w, http.StatusOK, result)
}


func (h *Handler) parseQueryParams(r *http.Request) *metrics.QueryRequest {
	query := r.URL.Query()
	
	queryType := query.Get("type")
	if queryType == "" {
		return nil
	}
	
	request := &metrics.QueryRequest{
		Query:     query.Get("query"),
		QueryType: metrics.QueryType(queryType),
		Filters:   make(map[string]string),
	}
	
	if startStr := query.Get("start"); startStr != "" {
		if start, err := time.Parse(time.RFC3339, startStr); err == nil {
			request.TimeRange.Start = start
		}
	}
	if endStr := query.Get("end"); endStr != "" {
		if end, err := time.Parse(time.RFC3339, endStr); err == nil {
			request.TimeRange.End = end
		}
	}
	
	for key, values := range query {
		if len(values) > 0 && !isReservedParam(key) {
			request.Filters[key] = values[0]
		}
	}
	
	if errorStr := query.Get("error_bound"); errorStr != "" {
		if error, err := strconv.ParseFloat(errorStr, 64); err == nil {
			request.ErrorBound = error
		}
	}
	if confStr := query.Get("confidence"); confStr != "" {
		if conf, err := strconv.ParseFloat(confStr, 64); err == nil {
			request.Confidence = conf
		}
	}
	
	return request
}

func isReservedParam(key string) bool {
	reserved := []string{"type", "query", "start", "end", "error_bound", "confidence"}
	for _, r := range reserved {
		if key == r {
			return true
		}
	}
	return false
}

func (h *Handler) generateTestMetrics(count int, clusterID, namespace string) {
	log.Printf("Generating %d test metrics for cluster: %s, namespace: %s", count, clusterID, namespace)
	
	metricNames := []string{"cpu_usage", "memory_usage", "disk_usage", "network_in", "network_out"}
	pods := []string{"pod-1", "pod-2", "pod-3", "pod-4", "pod-5"}
	
	for i := 0; i < count; i++ {
		metric := &metrics.MetricPoint{
			Timestamp:     time.Now(),
			ClusterID:     clusterID,
			Namespace:     namespace,
			PodName:       pods[i%len(pods)],
			ContainerName: "container-1",
			MetricName:    metricNames[i%len(metricNames)],
			Value:         rand.Float64(), // Random values between 0-1
			Unit:          "percent",
			Labels:        map[string]string{"generated": "true"},
		}
		
		h.queryEngine.ProcessMetric(metric)
		
		if i%1000 == 0 {
			log.Printf("Generated %d/%d test metrics", i, count)
		}
	}
	
	log.Printf("Completed generating %d test metrics", count)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string, err error) {
	errorResponse := map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	if err != nil {
		errorResponse["details"] = err.Error()
		log.Printf("API Error: %s - %v", message, err)
	}
	
	h.writeJSON(w, status, errorResponse)
}