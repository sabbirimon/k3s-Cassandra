package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Configuration
type Config struct {
	CassandraHost string
	Keyspace     string
	Datacenter   string
	Port         string
}

// Job model
type Job struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	AssignedTo  string    `json:"assigned_to"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Response models
type JobResponse struct {
	Job         interface{} `json:"job"`
	Pod         string      `json:"pod"`
	PodIP       string      `json:"pod_ip"`
	Database    string      `json:"database"`
	ClusterInfo ClusterInfo `json:"cluster_info"`
}

type ClusterInfo struct {
	ConnectionStatus string `json:"connection_status"`
	Language        string `json:"language"`
}

type JobsResponse struct {
	Jobs   []Job `json:"jobs"`
	Total  int   `json:"total"`
	Pod    string `json:"pod"`
	Language string `json:"language"`
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Pod       string    `json:"pod"`
	Timestamp string    `json:"timestamp"`
	Database  string    `json:"database"`
	Language  string    `json:"language"`
	Version   string    `json:"version"`
}

type InfoResponse struct {
	Service       string `json:"service"`
	Language      string `json:"language"`
	Framework     string `json:"framework"`
	Database      string `json:"database"`
	Version       string `json:"version"`
	Pod           string `json:"pod"`
	CassandraHost string `json:"cassandra_host"`
	Keyspace      string `json:"keyspace"`
	Datacenter    string `json:"datacenter"`
}

type ErrorResponse struct {
	Error string `json:"error"`
	Pod   string `json:"pod"`
}

type SuccessResponse struct {
	Message string `json:"message"`
	Pod     string `json:"pod"`
	Language string `json:"language"`
}

// Cassandra manager
type CassandraManager struct {
	cluster *gocql.ClusterConfig
	session *gocql.Session
	config  *Config
}

func NewCassandraManager(config *Config) *CassandraManager {
	return &CassandraManager{
		config: config,
	}
}

func (cm *CassandraManager) Connect() error {
	cluster := gocql.NewCluster(cm.config.CassandraHost)
	cluster.Keyspace = cm.config.Keyspace
	cluster.Consistency = gocql.Quorum
	cluster.ConnectTimeout = 30 * time.Second
	
	session, err := cluster.CreateSession()
	if err != nil {
		return fmt.Errorf("failed to connect to Cassandra: %v", err)
	}
	
	cm.session = session
	cm.cluster = cluster
	log.Printf("Connected to Cassandra cluster at %s", cm.config.CassandraHost)
	return nil
}

func (cm *CassandraManager) InitializeSchema() error {
	if cm.session == nil {
		return fmt.Errorf("session not initialized")
	}

	// Create keyspace
	err := cm.session.Query(fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s 
		WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 3}
	`, cm.config.Keyspace)).Exec()
	if err != nil {
		return fmt.Errorf("failed to create keyspace: %v", err)
	}

	// Create jobs table
	err = cm.session.Query(`
		CREATE TABLE IF NOT EXISTS jobs (
			id UUID PRIMARY KEY,
			title TEXT,
			description TEXT,
			status TEXT,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			assigned_to TEXT,
			priority INT
		)
	`).Exec()
	if err != nil {
		return fmt.Errorf("failed to create jobs table: %v", err)
	}

	// Insert sample data if table is empty
	var count int
	err = cm.session.Query("SELECT COUNT(*) FROM jobs").Consistency(gocql.One).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count jobs: %v", err)
	}

	if count == 0 {
		sampleJobs := []struct {
			Title       string
			Description string
			Status      string
			Priority    int
		}{
			{"Database Migration", "Migrate database to latest version", "pending", 1},
			{"API Development", "Develop REST API endpoints", "in_progress", 2},
			{"Testing Suite", "Create comprehensive test suite", "pending", 3},
			{"Documentation", "Write technical documentation", "pending", 4},
			{"Performance Optimization", "Optimize application performance", "pending", 5},
		}

		for _, job := range sampleJobs {
			err := cm.session.Query(`
				INSERT INTO jobs (id, title, description, status, created_at, updated_at, assigned_to, priority)
				VALUES (uuid(), ?, ?, ?, toTimestamp(now()), toTimestamp(now()), ?, ?)
			`, job.Title, job.Description, job.Status, "unassigned", job.Priority).Exec()
			if err != nil {
				return fmt.Errorf("failed to insert sample job: %v", err)
			}
		}
		log.Println("Sample jobs inserted into Cassandra")
	}

	log.Println("Cassandra schema initialization completed")
	return nil
}

func (cm *CassandraManager) GetRandomJob() (*Job, error) {
	if cm.session == nil {
		return nil, fmt.Errorf("session not initialized")
	}

	var job Job
	iter := cm.session.Query("SELECT * FROM jobs LIMIT 5").Iter()
	defer iter.Close()

	var jobs []Job
	for iter.Scan(&job.ID, &job.Title, &job.Description, &job.Status, 
		&job.CreatedAt, &job.UpdatedAt, &job.AssignedTo, &job.Priority) {
		jobs = append(jobs, job)
	}

	if len(jobs) == 0 {
		return nil, nil
	}

	// Return first job (simplified random selection)
	return &jobs[0], nil
}

func (cm *CassandraManager) GetAllJobs() ([]Job, error) {
	if cm.session == nil {
		return nil, fmt.Errorf("session not initialized")
	}

	var jobs []Job
	iter := cm.session.Query("SELECT * FROM jobs").Iter()
	defer iter.Close()

	var job Job
	for iter.Scan(&job.ID, &job.Title, &job.Description, &job.Status, 
		&job.CreatedAt, &job.UpdatedAt, &job.AssignedTo, &job.Priority) {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (cm *CassandraManager) CreateJob(title, description, status, assignedTo string, priority int) error {
	if cm.session == nil {
		return fmt.Errorf("session not initialized")
	}

	return cm.session.Query(`
		INSERT INTO jobs (id, title, description, status, created_at, updated_at, assigned_to, priority)
		VALUES (uuid(), ?, ?, ?, toTimestamp(now()), toTimestamp(now()), ?, ?)
	`, title, description, status, assignedTo, priority).Exec()
}

func (cm *CassandraManager) Close() {
	if cm.session != nil {
		cm.session.Close()
		log.Println("Cassandra connection closed")
	}
}

// HTTP handlers
func (cm *CassandraManager) getRandomJobHandler(w http.ResponseWriter, r *http.Request) {
	job, err := cm.GetRandomJob()
	if err != nil {
		log.Printf("Error getting random job: %v", err)
		http.Error(w, "Failed to fetch job from database", http.StatusInternalServerError)
		return
	}

	hostname, _ := os.Hostname()
	response := JobResponse{
		Job:   job,
		Pod:   hostname,
		PodIP: r.Host,
		Database: "Cassandra",
		ClusterInfo: ClusterInfo{
			ConnectionStatus: "connected",
			Language:        "Go",
		},
	}

	if job == nil {
		response.Job = map[string]interface{}{
			"title":       "No jobs found",
			"description": "Database is empty",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (cm *CassandraManager) getAllJobsHandler(w http.ResponseWriter, r *http.Request) {
	jobs, err := cm.GetAllJobs()
	if err != nil {
		log.Printf("Error getting all jobs: %v", err)
		http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
		return
	}

	hostname, _ := os.Hostname()
	response := JobsResponse{
		Jobs:    jobs,
		Total:   len(jobs),
		Pod:     hostname,
		Language: "Go",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (cm *CassandraManager) createJobHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		AssignedTo  string `json:"assigned_to"`
		Priority    int    `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Title == "" || req.Description == "" {
		http.Error(w, "Title and description are required", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		req.Status = "pending"
	}
	if req.AssignedTo == "" {
		req.AssignedTo = "unassigned"
	}
	if req.Priority == 0 {
		req.Priority = 1
	}

	err := cm.CreateJob(req.Title, req.Description, req.Status, req.AssignedTo, req.Priority)
	if err != nil {
		log.Printf("Error creating job: %v", err)
		http.Error(w, "Failed to create job", http.StatusInternalServerError)
		return
	}

	hostname, _ := os.Hostname()
	response := SuccessResponse{
		Message: "Job created successfully",
		Pod:     hostname,
		Language: "Go",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (cm *CassandraManager) healthHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	
	// Test Cassandra connection
	job, err := cm.GetRandomJob()
	dbStatus := "connected"
	if err != nil {
		dbStatus = "disconnected"
	}

	response := HealthResponse{
		Status:    "healthy",
		Pod:       hostname,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Database:  dbStatus,
		Language:  "Go",
		Version:   "1.0.0",
	}

	if err != nil {
		response.Status = "unhealthy"
		response.Database = "disconnected"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (cm *CassandraManager) infoHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	
	response := InfoResponse{
		Service:       "Backend API",
		Language:      "Go",
		Framework:     "Gorilla Mux",
		Database:      "Cassandra",
		Version:       "1.0.0",
		Pod:           hostname,
		CassandraHost: cm.config.CassandraHost,
		Keyspace:      cm.config.Keyspace,
		Datacenter:    cm.config.Datacenter,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	config := &Config{
		CassandraHost: getEnv("CASSANDRA_HOST", "cassandra.cassandra.svc.cluster.local"),
		Keyspace:     getEnv("CASSANDRA_KEYSPACE", "job_tracker"),
		Datacenter:   getEnv("CASSANDRA_DC", "datacenter1"),
		Port:         getEnv("PORT", "5000"),
	}

	// Initialize Cassandra
	cassandraManager := NewCassandraManager(config)
	if err := cassandraManager.Connect(); err != nil {
		log.Fatalf("Failed to connect to Cassandra: %v", err)
	}

	if err := cassandraManager.InitializeSchema(); err != nil {
		log.Fatalf("Failed to initialize Cassandra schema: %v", err)
	}

	// Setup router
	router := mux.NewRouter()
	
	// API routes
	router.HandleFunc("/", cassandraManager.getRandomJobHandler).Methods("GET")
	router.HandleFunc("/jobs", cassandraManager.getAllJobsHandler).Methods("GET")
	router.HandleFunc("/jobs", cassandraManager.createJobHandler).Methods("POST")
	router.HandleFunc("/health", cassandraManager.healthHandler).Methods("GET")
	router.HandleFunc("/info", cassandraManager.infoHandler).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Start server
	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: handler,
	}

	log.Printf("Starting Go backend server on port %s", config.Port)
	log.Printf("Cassandra host: %s", config.CassandraHost)
	log.Printf("Keyspace: %s", config.Keyspace)

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	cassandraManager.Close()
	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}