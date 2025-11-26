from flask import Flask, render_template, request, jsonify
import requests
import os
import socket
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = Flask(__name__)

# Configuration
BACKEND_URL = os.getenv('BACKEND_URL', 'http://backend-service:5000')
PORT = int(os.getenv('PORT', 8080))

class BackendService:
    def __init__(self, base_url):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.timeout = 10
    
    def get_random_job(self):
        try:
            response = self.session.get(f"{self.base_url}/")
            response.raise_for_status()
            return response.json()
        except Exception as e:
            logger.error(f"Error fetching random job: {e}")
            return None
    
    def get_all_jobs(self):
        try:
            response = self.session.get(f"{self.base_url}/jobs")
            response.raise_for_status()
            return response.json()
        except Exception as e:
            logger.error(f"Error fetching all jobs: {e}")
            return None
    
    def create_job(self, job_data):
        try:
            response = self.session.post(f"{self.base_url}/jobs", json=job_data)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            logger.error(f"Error creating job: {e}")
            return None
    
    def get_health(self):
        try:
            response = self.session.get(f"{self.base_url}/health")
            response.raise_for_status()
            return response.json()
        except Exception as e:
            logger.error(f"Error checking backend health: {e}")
            return None

backend_service = BackendService(BACKEND_URL)

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/api/job')
def api_get_job():
    data = backend_service.get_random_job()
    if data:
        return jsonify(data)
    return jsonify({'error': 'Failed to fetch job from backend'}), 500

@app.route('/api/jobs')
def api_get_jobs():
    data = backend_service.get_all_jobs()
    if data:
        return jsonify(data)
    return jsonify({'error': 'Failed to fetch jobs from backend'}), 500

@app.route('/api/jobs', methods=['POST'])
def api_create_job():
    job_data = request.get_json()
    if not job_data:
        return jsonify({'error': 'No job data provided'}), 400
    
    data = backend_service.create_job(job_data)
    if data:
        return jsonify(data)
    return jsonify({'error': 'Failed to create job'}), 500

@app.route('/api/health')
def api_health():
    data = backend_service.get_health()
    if data:
        return jsonify(data)
    return jsonify({'error': 'Backend service unavailable'}), 500

@app.route('/api/info')
def api_info():
    hostname = socket.gethostname()
    return jsonify({
        'service': 'Frontend Application',
        'language': 'Python',
        'framework': 'Flask',
        'version': '1.0.0',
        'pod': hostname,
        'backend_url': BACKEND_URL
    })

if __name__ == '__main__':
    logger.info(f"Starting Python frontend server on port {PORT}")
    logger.info(f"Backend URL: {BACKEND_URL}")
    
    app.run(host='0.0.0.0', port=PORT, debug=False)