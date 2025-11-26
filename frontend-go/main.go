package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Configuration
type Config struct {
	BackendURL string
	Port       string
}

// Backend service
type BackendService struct {
	BaseURL string
	Client  *http.Client
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
	Jobs    []Job `json:"jobs"`
	Total   int   `json:"total"`
	Pod     string `json:"pod"`
	Language string `json:"language"`
}

type HealthResponse struct {
	Status   string `json:"status"`
	Pod      string `json:"pod"`
	Timestamp string `json:"timestamp"`
	Database string `json:"database"`
	Language string `json:"language"`
	Version  string `json:"version"`
}

type InfoResponse struct {
	Service    string `json:"service"`
	Language   string `json:"language"`
	Framework  string `json:"framework"`
	Version    string `json:"version"`
	Pod        string `json:"pod"`
	BackendURL string `json:"backend_url"`
}

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

func NewBackendService(baseURL string) *BackendService {
	return &BackendService{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (bs *BackendService) GetRandomJob() (*JobResponse, error) {
	resp, err := bs.Client.Get(bs.BaseURL + "/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch random job: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	var jobResp JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &jobResp, nil
}

func (bs *BackendService) GetAllJobs() (*JobsResponse, error) {
	resp, err := bs.Client.Get(bs.BaseURL + "/jobs")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all jobs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	var jobsResp JobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &jobsResp, nil
}

func (bs *BackendService) CreateJob(jobData map[string]interface{}) (*map[string]interface{}, error) {
	jsonData, err := json.Marshal(jobData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job data: %v", err)
	}

	resp, err := bs.Client.Post(bs.BaseURL+"/jobs", "application/json", 
		&jsonDataBuffer{jsonData})
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}

func (bs *BackendService) GetHealth() (*HealthResponse, error) {
	resp, err := bs.Client.Get(bs.BaseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch health status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &healthResp, nil
}

// Helper type for http.Client.Post
type jsonDataBuffer struct {
	data []byte
}

func (j *jsonDataBuffer) Read(p []byte) (n int, err error) {
	if len(j.data) == 0 {
		return 0, fmt.Errorf("no data")
	}
	n = copy(p, j.data)
	j.data = j.data[n:]
	if len(j.data) == 0 {
		err = fmt.Errorf("EOF")
	}
	return n, nil
}

// HTTP handlers
func (bs *BackendService) apiGetJobHandler(w http.ResponseWriter, r *http.Request) {
	data, err := bs.GetRandomJob()
	if err != nil {
		log.Printf("Error fetching random job: %v", err)
		http.Error(w, "Failed to fetch job from backend", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (bs *BackendService) apiGetJobsHandler(w http.ResponseWriter, r *http.Request) {
	data, err := bs.GetAllJobs()
	if err != nil {
		log.Printf("Error fetching all jobs: %v", err)
		http.Error(w, "Failed to fetch jobs from backend", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (bs *BackendService) apiCreateJobHandler(w http.ResponseWriter, r *http.Request) {
	var jobData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&jobData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	data, err := bs.CreateJob(jobData)
	if err != nil {
		log.Printf("Error creating job: %v", err)
		http.Error(w, "Failed to create job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (bs *BackendService) apiHealthHandler(w http.ResponseWriter, r *http.Request) {
	data, err := bs.GetHealth()
	if err != nil {
		log.Printf("Error checking backend health: %v", err)
		http.Error(w, "Backend service unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (bs *BackendService) apiInfoHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	
	response := InfoResponse{
		Service:    "Frontend Application",
		Language:   "Go",
		Framework:  "Gorilla Mux",
		Version:    "1.0.0",
		Pod:        hostname,
		BackendURL: bs.BaseURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (bs *BackendService) indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("index").Parse(indexHTML))
	
	data := map[string]interface{}{
		"Title": "Kubernetes Networking Demo - Go Frontend",
		"Language": "Go",
		"Framework": "Gorilla Mux",
	}
	
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HTML template
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #00c9ff 0%, #92fe9d 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #2c3e50 0%, #34495e 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 15px;
        }
        
        .header p {
            font-size: 1.1em;
            opacity: 0.9;
        }
        
        .main-content {
            padding: 40px;
        }
        
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 25px;
            margin-bottom: 30px;
        }
        
        .info-card {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 25px;
            border-left: 5px solid #00c9ff;
            transition: transform 0.3s ease, box-shadow 0.3s ease;
        }
        
        .info-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
        }
        
        .info-card h3 {
            color: #2c3e50;
            margin-bottom: 15px;
            font-size: 1.3em;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        
        .info-item {
            margin: 10px 0;
            padding: 8px 0;
            border-bottom: 1px solid #e9ecef;
        }
        
        .info-item:last-child {
            border-bottom: none;
        }
        
        .label {
            font-weight: 600;
            color: #495057;
            display: inline-block;
            min-width: 120px;
        }
        
        .value {
            color: #00c9ff;
            font-family: 'Courier New', monospace;
            background: #e3f2fd;
            padding: 2px 8px;
            border-radius: 4px;
        }
        
        .status {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 0.85em;
            font-weight: 600;
        }
        
        .status.healthy {
            background: #d4edda;
            color: #155724;
        }
        
        .status.error {
            background: #f8d7da;
            color: #721c24;
        }
        
        .loading {
            text-align: center;
            padding: 40px;
            color: #6c757d;
        }
        
        .loading::after {
            content: '';
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 3px solid #f3f3f3;
            border-top: 3px solid #00c9ff;
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin-left: 10px;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .button-group {
            display: flex;
            gap: 15px;
            justify-content: center;
            margin: 30px 0;
            flex-wrap: wrap;
        }
        
        .btn {
            background: linear-gradient(135deg, #00c9ff 0%, #92fe9d 100%);
            color: white;
            border: none;
            padding: 12px 25px;
            border-radius: 8px;
            cursor: pointer;
            font-size: 1em;
            font-weight: 600;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(0,201,255,0.3);
        }
        
        .btn.secondary {
            background: linear-gradient(135deg, #95a5a6 0%, #7f8c8d 100%);
        }
        
        .btn.success {
            background: linear-gradient(135deg, #27ae60 0%, #229954 100%);
        }
        
        .jobs-section {
            margin-top: 40px;
            padding: 30px;
            background: #f8f9fa;
            border-radius: 10px;
        }
        
        .jobs-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 20px;
            margin-top: 20px;
        }
        
        .job-card {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            border-left: 4px solid #27ae60;
        }
        
        .job-title {
            font-weight: 600;
            color: #2c3e50;
            margin-bottom: 10px;
        }
        
        .job-description {
            color: #6c757d;
            margin-bottom: 15px;
        }
        
        .job-meta {
            display: flex;
            justify-content: space-between;
            font-size: 0.9em;
        }
        
        .error-message {
            background: #f8d7da;
            color: #721c24;
            padding: 15px;
            border-radius: 8px;
            margin: 20px 0;
            border-left: 4px solid #dc3545;
        }
        
        .footer {
            background: #2c3e50;
            color: white;
            text-align: center;
            padding: 20px;
            margin-top: 40px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>
                🚀 Kubernetes Networking Demo
                <span style="font-size: 0.6em; background: #e74c3c; padding: 5px 10px; border-radius: 15px;">Go Stack</span>
            </h1>
            <p>Multi-tier application demonstrating Kubernetes Services with Go + Gorilla Mux + Cassandra</p>
        </div>
        
        <div class="main-content">
            <div id="loading" class="loading">Loading application data...</div>
            <div id="error" class="error-message" style="display: none;"></div>
            
            <div id="content" style="display: none;">
                <div class="info-grid">
                    <div class="info-card">
                        <h3>🎯 Current Job</h3>
                        <div id="job-info">
                            <div class="info-item">
                                <span class="label">Title:</span>
                                <span class="value" id="job-title">-</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Description:</span>
                                <span class="value" id="job-description">-</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Status:</span>
                                <span class="value" id="job-status">-</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Priority:</span>
                                <span class="value" id="job-priority">-</span>
                            </div>
                        </div>
                    </div>
                    
                    <div class="info-card">
                        <h3>🔧 Backend Service</h3>
                        <div id="backend-info">
                            <div class="info-item">
                                <span class="label">Pod:</span>
                                <span class="value" id="backend-pod">-</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Pod IP:</span>
                                <span class="value" id="backend-ip">-</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Database:</span>
                                <span class="value" id="database-type">-</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Language:</span>
                                <span class="value" id="backend-language">-</span>
                            </div>
                        </div>
                    </div>
                    
                    <div class="info-card">
                        <h3>🌐 Frontend Service</h3>
                        <div id="frontend-info">
                            <div class="info-item">
                                <span class="label">Pod:</span>
                                <span class="value" id="frontend-pod">-</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Client IP:</span>
                                <span class="value" id="client-ip">-</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Framework:</span>
                                <span class="value" id="frontend-framework">{{.Framework}}</span>
                            </div>
                            <div class="info-item">
                                <span class="label">Health Status:</span>
                                <span class="status" id="health-status">-</span>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div class="button-group">
                    <button class="btn" onclick="refreshData()">
                        🔄 Refresh Data
                    </button>
                    <button class="btn secondary" onclick="showAllJobs()">
                        📋 View All Jobs
                    </button>
                    <button class="btn success" onclick="createNewJob()">
                        ➕ Create New Job
                    </button>
                </div>
                
                <div id="jobs-section" class="jobs-section" style="display: none;">
                    <h3>📊 All Jobs in Database</h3>
                    <div id="jobs-grid" class="jobs-grid"></div>
                </div>
            </div>
        </div>
        
        <div class="footer">
            <p>Kubernetes Networking Lab | Go + Gorilla Mux + Cassandra | Multi-Language Demo</p>
        </div>
    </div>

    <script>
        let currentData = {};
        
        async function fetchBackendData() {
            try {
                const response = await fetch('/api/job');
                if (!response.ok) throw new Error(\`HTTP \${response.status}: \${response.statusText}\`);
                return await response.json();
            } catch (error) {
                console.error('Error fetching backend data:', error);
                throw error;
            }
        }
        
        async function fetchAllJobs() {
            try {
                const response = await fetch('/api/jobs');
                if (!response.ok) throw new Error(\`HTTP \${response.status}: \${response.statusText}\`);
                return await response.json();
            } catch (error) {
                console.error('Error fetching all jobs:', error);
                throw error;
            }
        }
        
        async function fetchHealthStatus() {
            try {
                const response = await fetch('/api/health');
                if (!response.ok) throw new Error(\`HTTP \${response.status}: \${response.statusText}\`);
                return await response.json();
            } catch (error) {
                console.error('Error fetching health status:', error);
                throw error;
            }
        }
        
        function updateJobInfo(job) {
            if (job && job.title) {
                document.getElementById('job-title').textContent = job.title;
                document.getElementById('job-description').textContent = job.description || 'No description';
                document.getElementById('job-status').textContent = job.status || 'unknown';
                document.getElementById('job-priority').textContent = job.priority || 'N/A';
            } else {
                document.getElementById('job-title').textContent = 'No jobs available';
                document.getElementById('job-description').textContent = 'Database is empty';
                document.getElementById('job-status').textContent = 'N/A';
                document.getElementById('job-priority').textContent = 'N/A';
            }
        }
        
        function updateBackendInfo(data) {
            document.getElementById('backend-pod').textContent = data.pod || 'Unknown';
            document.getElementById('backend-ip').textContent = data.podIP || 'Unknown';
            document.getElementById('database-type').textContent = data.database || 'Unknown';
            document.getElementById('backend-language').textContent = data.cluster_info?.language || 'Unknown';
        }
        
        function updateFrontendInfo() {
            document.getElementById('frontend-pod').textContent = 'Frontend Pod';
            document.getElementById('client-ip').textContent = window.location.hostname;
        }
        
        function updateHealthStatus(health) {
            const statusElement = document.getElementById('health-status');
            if (health.status === 'healthy') {
                statusElement.textContent = '✅ Healthy';
                statusElement.className = 'status healthy';
            } else {
                statusElement.textContent = '❌ Unhealthy';
                statusElement.className = 'status error';
            }
        }
        
        function displayAllJobs(jobsData) {
            const jobsSection = document.getElementById('jobs-section');
            const jobsGrid = document.getElementById('jobs-grid');
            
            if (!jobsData.jobs || jobsData.jobs.length === 0) {
                jobsGrid.innerHTML = '<p>No jobs found in database.</p>';
            } else {
                jobsGrid.innerHTML = jobsData.jobs.map(job => \`
                    <div class="job-card">
                        <div class="job-title">\${job.title}</div>
                        <div class="job-description">\${job.description || 'No description'}</div>
                        <div class="job-meta">
                            <span>Status: <strong>\${job.status || 'unknown'}</strong></span>
                            <span>Priority: <strong>\${job.priority || 'N/A'}</strong></span>
                        </div>
                    </div>
                \`).join('');
            }
            
            jobsSection.style.display = 'block';
        }
        
        async function refreshData() {
            const loading = document.getElementById('loading');
            const content = document.getElementById('content');
            const error = document.getElementById('error');
            
            loading.style.display = 'block';
            content.style.display = 'none';
            error.style.display = 'none';
            
            try {
                // Fetch all data in parallel
                const [backendData, allJobs, healthStatus] = await Promise.all([
                    fetchBackendData(),
                    fetchAllJobs(),
                    fetchHealthStatus()
                ]);
                
                currentData = { backendData, allJobs, healthStatus };
                
                // Update UI
                updateJobInfo(backendData.job);
                updateBackendInfo(backendData);
                updateFrontendInfo();
                updateHealthStatus(healthStatus);
                
                loading.style.display = 'none';
                content.style.display = 'block';
                
            } catch (error) {
                loading.style.display = 'none';
                error.style.display = 'block';
                error.innerHTML = \`
                    <strong>Error:</strong> \${error.message}<br>
                    <small>Please check if backend service is running and accessible.</small>
                \`;
            }
        }
        
        function showAllJobs() {
            if (currentData.allJobs) {
                displayAllJobs(currentData.allJobs);
            } else {
                alert('Please refresh data first');
            }
        }
        
        function createNewJob() {
            const title = prompt('Enter job title:');
            if (!title) return;
            
            const description = prompt('Enter job description:');
            if (!description) return;
            
            const status = prompt('Enter job status (pending/in_progress/completed):', 'pending');
            const priority = prompt('Enter job priority (1-5):', '1');
            
            fetch('/api/jobs', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    title,
                    description,
                    status: status || 'pending',
                    priority: parseInt(priority) || 1
                })
            })
            .then(response => response.json())
            .then(data => {
                alert('Job created successfully!');
                refreshData();
            })
            .catch(error => {
                alert('Error creating job: ' + error.message);
            });
        }
        
        // Initialize on page load
        document.addEventListener('DOMContentLoaded', refreshData);
        
        // Auto-refresh every 30 seconds
        setInterval(refreshData, 30000);
    </script>
</body>
</html>`

func main() {
	config := &Config{
		BackendURL: getEnv("BACKEND_URL", "http://backend-service:5000"),
		Port:       getEnv("PORT", "8080"),
	}

	// Initialize backend service
	backendService := NewBackendService(config.BackendURL)

	// Setup router
	router := mux.NewRouter()
	
	// API routes
	router.HandleFunc("/api/job", backendService.apiGetJobHandler).Methods("GET")
	router.HandleFunc("/api/jobs", backendService.apiGetJobsHandler).Methods("GET")
	router.HandleFunc("/api/jobs", backendService.apiCreateJobHandler).Methods("POST")
	router.HandleFunc("/api/health", backendService.apiHealthHandler).Methods("GET")
	router.HandleFunc("/api/info", backendService.apiInfoHandler).Methods("GET")
	
	// Main page
	router.HandleFunc("/", backendService.indexHandler).Methods("GET")

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

	log.Printf("Starting Go frontend server on port %s", config.Port)
	log.Printf("Backend URL: %s", config.BackendURL)

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

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}