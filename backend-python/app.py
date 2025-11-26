from flask import Flask, jsonify, request
from flask_cors import CORS
import cassandra.cluster
import cassandra.util
import uuid
import os
import socket
import datetime
import logging
from typing import Dict, List, Optional

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)

# Configuration
CASSANDRA_HOST = os.getenv('CASSANDRA_HOST', 'cassandra.cassandra.svc.cluster.local')
CASSANDRA_KEYSPACE = os.getenv('CASSANDRA_KEYSPACE', 'job_tracker')
CASSANDRA_DC = os.getenv('CASSANDRA_DC', 'datacenter1')
PORT = int(os.getenv('PORT', 5000))

# Global Cassandra session
session = None

class CassandraManager:
    def __init__(self, contact_points: List[str], keyspace: str):
        self.contact_points = contact_points
        self.keyspace = keyspace
        self.cluster = None
        self.session = None
    
    def connect(self):
        try:
            self.cluster = cassandra.cluster.Cluster(
                contact_points=self.contact_points,
                load_balancing_policy=cassandra.cluster.DCAwareRoundRobinPolicy(local_dc=CASSANDRA_DC)
            )
            self.session = self.cluster.connect()
            logger.info(f"Connected to Cassandra cluster at {self.contact_points}")
            return True
        except Exception as e:
            logger.error(f"Failed to connect to Cassandra: {e}")
            return False
    
    def initialize_schema(self):
        try:
            # Create keyspace
            self.session.execute(f"""
                CREATE KEYSPACE IF NOT EXISTS {self.keyspace}
                WITH replication = {{'class': 'SimpleStrategy', 'replication_factor': 3}}
            """)
            
            # Use keyspace
            self.session.execute(f"USE {self.keyspace}")
            
            # Create jobs table
            self.session.execute("""
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
            """)
            
            # Insert sample data if table is empty
            result = self.session.execute("SELECT COUNT(*) FROM jobs")
            if result[0].count == 0:
                sample_jobs = [
                    ('Database Migration', 'Migrate database to latest version', 'pending', 1),
                    ('API Development', 'Develop REST API endpoints', 'in_progress', 2),
                    ('Testing Suite', 'Create comprehensive test suite', 'pending', 3),
                    ('Documentation', 'Write technical documentation', 'pending', 4),
                    ('Performance Optimization', 'Optimize application performance', 'pending', 5)
                ]
                
                for title, description, status, priority in sample_jobs:
                    self.session.execute("""
                        INSERT INTO jobs (id, title, description, status, created_at, updated_at, assigned_to, priority)
                        VALUES (uuid(), %s, %s, %s, toTimestamp(now()), toTimestamp(now()), %s, %s)
                    """, (title, description, status, 'unassigned', priority))
                
                logger.info("Sample jobs inserted into Cassandra")
            
            logger.info("Cassandra schema initialization completed")
            return True
        except Exception as e:
            logger.error(f"Failed to initialize schema: {e}")
            return False
    
    def get_random_job(self) -> Optional[Dict]:
        try:
            result = self.session.execute("SELECT * FROM jobs LIMIT 5")
            if result:
                job = result[0]
                return {
                    'id': str(job.id),
                    'title': job.title,
                    'description': job.description,
                    'status': job.status,
                    'priority': job.priority,
                    'assigned_to': job.assigned_to,
                    'created_at': job.created_at.isoformat() if job.created_at else None,
                    'updated_at': job.updated_at.isoformat() if job.updated_at else None
                }
            return None
        except Exception as e:
            logger.error(f"Failed to get random job: {e}")
            return None
    
    def get_all_jobs(self) -> List[Dict]:
        try:
            result = self.session.execute("SELECT * FROM jobs")
            jobs = []
            for job in result:
                jobs.append({
                    'id': str(job.id),
                    'title': job.title,
                    'description': job.description,
                    'status': job.status,
                    'priority': job.priority,
                    'assigned_to': job.assigned_to,
                    'created_at': job.created_at.isoformat() if job.created_at else None,
                    'updated_at': job.updated_at.isoformat() if job.updated_at else None
                })
            return jobs
        except Exception as e:
            logger.error(f"Failed to get all jobs: {e}")
            return []
    
    def create_job(self, title: str, description: str, status: str = 'pending', 
                  assigned_to: str = 'unassigned', priority: int = 1) -> bool:
        try:
            self.session.execute("""
                INSERT INTO jobs (id, title, description, status, created_at, updated_at, assigned_to, priority)
                VALUES (uuid(), %s, %s, %s, toTimestamp(now()), toTimestamp(now()), %s, %s)
            """, (title, description, status, assigned_to, priority))
            return True
        except Exception as e:
            logger.error(f"Failed to create job: {e}")
            return False
    
    def shutdown(self):
        if self.cluster:
            self.cluster.shutdown()
            logger.info("Cassandra connection closed")

# Initialize Cassandra
cassandra_manager = CassandraManager([CASSANDRA_HOST], CASSANDRA_KEYSPACE)

@app.before_first_request
def initialize_cassandra():
    if not cassandra_manager.connect():
        logger.error("Failed to connect to Cassandra on startup")
        return False
    
    if not cassandra_manager.initialize_schema():
        logger.error("Failed to initialize Cassandra schema")
        return False
    
    logger.info("Cassandra initialization completed successfully")
    return True

@app.route('/')
def get_random_job():
    try:
        job = cassandra_manager.get_random_job()
        if job:
            return jsonify({
                'job': job,
                'pod': socket.gethostname(),
                'pod_ip': request.host.split(':')[0],
                'database': 'Cassandra',
                'cluster_info': {
                    'connection_status': 'connected',
                    'language': 'Python'
                }
            })
        else:
            return jsonify({
                'job': {'title': 'No jobs found', 'description': 'Database is empty'},
                'pod': socket.gethostname(),
                'pod_ip': request.host.split(':')[0],
                'database': 'Cassandra',
                'cluster_info': {
                    'connection_status': 'connected',
                    'language': 'Python'
                }
            })
    except Exception as e:
        logger.error(f"Error in get_random_job: {e}")
        return jsonify({
            'error': 'Failed to fetch job from database',
            'pod': socket.gethostname(),
            'pod_ip': request.host.split(':')[0]
        }), 500

@app.route('/jobs')
def get_all_jobs():
    try:
        jobs = cassandra_manager.get_all_jobs()
        return jsonify({
            'jobs': jobs,
            'total': len(jobs),
            'pod': socket.gethostname(),
            'language': 'Python'
        })
    except Exception as e:
        logger.error(f"Error in get_all_jobs: {e}")
        return jsonify({'error': 'Failed to fetch jobs'}), 500

@app.route('/jobs', methods=['POST'])
def create_job():
    try:
        data = request.get_json()
        
        if not data or not data.get('title') or not data.get('description'):
            return jsonify({'error': 'Title and description are required'}), 400
        
        title = data['title']
        description = data['description']
        status = data.get('status', 'pending')
        assigned_to = data.get('assigned_to', 'unassigned')
        priority = data.get('priority', 1)
        
        if cassandra_manager.create_job(title, description, status, assigned_to, priority):
            return jsonify({
                'message': 'Job created successfully',
                'pod': socket.gethostname(),
                'language': 'Python'
            })
        else:
            return jsonify({'error': 'Failed to create job'}), 500
    except Exception as e:
        logger.error(f"Error in create_job: {e}")
        return jsonify({'error': 'Failed to create job'}), 500

@app.route('/health')
def health_check():
    try:
        # Test Cassandra connection
        job = cassandra_manager.get_random_job()
        db_status = 'connected' if job is not None else 'disconnected'
        
        return jsonify({
            'status': 'healthy',
            'pod': socket.gethostname(),
            'timestamp': datetime.datetime.utcnow().isoformat(),
            'database': db_status,
            'language': 'Python',
            'version': '1.0.0'
        })
    except Exception as e:
        logger.error(f"Error in health_check: {e}")
        return jsonify({
            'status': 'unhealthy',
            'pod': socket.gethostname(),
            'timestamp': datetime.datetime.utcnow().isoformat(),
            'database': 'disconnected',
            'language': 'Python',
            'error': str(e)
        }), 500

@app.route('/info')
def get_info():
    return jsonify({
        'service': 'Backend API',
        'language': 'Python',
        'framework': 'Flask',
        'database': 'Cassandra',
        'version': '1.0.0',
        'pod': socket.gethostname(),
        'cassandra_host': CASSANDRA_HOST,
        'keyspace': CASSANDRA_KEYSPACE,
        'datacenter': CASSANDRA_DC
    })

if __name__ == '__main__':
    # Initialize Cassandra before starting the server
    if not cassandra_manager.connect():
        logger.error("Failed to connect to Cassandra. Exiting...")
        exit(1)
    
    if not cassandra_manager.initialize_schema():
        logger.error("Failed to initialize Cassandra schema. Exiting...")
        exit(1)
    
    logger.info(f"Starting Python backend server on port {PORT}")
    logger.info(f"Cassandra host: {CASSANDRA_HOST}")
    logger.info(f"Keyspace: {CASSANDRA_KEYSPACE}")
    
    try:
        app.run(host='0.0.0.0', port=PORT, debug=False)
    except KeyboardInterrupt:
        logger.info("Shutting down gracefully...")
    finally:
        cassandra_manager.shutdown()