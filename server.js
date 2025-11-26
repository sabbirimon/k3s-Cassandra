const express = require('express');
const { Client } = require('cassandra-driver');
const os = require('os');
const cors = require('cors');
const app = express();
const PORT = 5000;

app.use(cors());
app.use(express.json());

// Cassandra connection configuration
const cassandraClient = new Client({
  contactPoints: [process.env.CASSANDRA_HOST || 'cassandra.cassandra.svc.cluster.local'],
  localDataCenter: process.env.CASSANDRA_DC || 'datacenter1',
  keyspace: process.env.CASSANDRA_KEYSPACE || 'job_tracker'
});

// Initialize Cassandra connection and keyspace
async function initializeCassandra() {
  try {
    await cassandraClient.connect();
    console.log('Connected to Cassandra cluster');
    
    // Create keyspace if it doesn't exist
    await cassandraClient.execute(`
      CREATE KEYSPACE IF NOT EXISTS job_tracker 
      WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 3}
    `);
    
    // Use the keyspace
    await cassandraClient.execute('USE job_tracker');
    
    // Create jobs table if it doesn't exist
    await cassandraClient.execute(`
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
    `);
    
    // Insert sample jobs if table is empty
    const result = await cassandraClient.execute('SELECT COUNT(*) FROM jobs');
    if (result.rows[0].count === 0) {
      const sampleJobs = [
        { title: 'Database Migration', description: 'Migrate database to latest version', status: 'pending', priority: 1 },
        { title: 'API Development', description: 'Develop REST API endpoints', status: 'in_progress', priority: 2 },
        { title: 'Testing Suite', description: 'Create comprehensive test suite', status: 'pending', priority: 3 },
        { title: 'Documentation', description: 'Write technical documentation', status: 'pending', priority: 4 },
        { title: 'Performance Optimization', description: 'Optimize application performance', status: 'pending', priority: 5 }
      ];
      
      for (const job of sampleJobs) {
        await cassandraClient.execute(`
          INSERT INTO jobs (id, title, description, status, created_at, updated_at, assigned_to, priority)
          VALUES (uuid(), ?, ?, ?, toTimestamp(now()), toTimestamp(now()), ?, ?)
        `, [job.title, job.description, job.status, 'unassigned', job.priority]);
      }
      
      console.log('Sample jobs inserted into Cassandra');
    }
    
    console.log('Cassandra initialization completed');
  } catch (error) {
    console.error('Failed to initialize Cassandra:', error);
    process.exit(1);
  }
}

// API Routes
app.get('/', async (req, res) => {
  try {
    const result = await cassandraClient.execute('SELECT * FROM jobs LIMIT 5');
    const randomJob = result.rows[Math.floor(Math.random() * result.rows.length)];
    
    res.json({
      job: randomJob || { title: 'No jobs found', description: 'Database is empty' },
      pod: os.hostname(),
      podIP: req.connection.localAddress,
      database: 'Cassandra',
      cluster_info: {
        total_jobs: result.rowLength,
        connection_status: 'connected'
      }
    });
  } catch (error) {
    console.error('Error fetching job:', error);
    res.status(500).json({
      error: 'Failed to fetch job from database',
      pod: os.hostname(),
      podIP: req.connection.localAddress
    });
  }
});

app.get('/jobs', async (req, res) => {
  try {
    const result = await cassandraClient.execute('SELECT * FROM jobs');
    res.json({
      jobs: result.rows,
      total: result.rowLength,
      pod: os.hostname()
    });
  } catch (error) {
    console.error('Error fetching jobs:', error);
    res.status(500).json({ error: 'Failed to fetch jobs' });
  }
});

app.post('/jobs', async (req, res) => {
  try {
    const { title, description, status = 'pending', assigned_to = 'unassigned', priority = 1 } = req.body;
    
    if (!title || !description) {
      return res.status(400).json({ error: 'Title and description are required' });
    }
    
    await cassandraClient.execute(`
      INSERT INTO jobs (id, title, description, status, created_at, updated_at, assigned_to, priority)
      VALUES (uuid(), ?, ?, ?, toTimestamp(now()), toTimestamp(now()), ?, ?)
    `, [title, description, status, assigned_to, priority]);
    
    res.json({ 
      message: 'Job created successfully',
      pod: os.hostname()
    });
  } catch (error) {
    console.error('Error creating job:', error);
    res.status(500).json({ error: 'Failed to create job' });
  }
});

app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    pod: os.hostname(),
    timestamp: new Date().toISOString(),
    database: cassandraClient.connected ? 'connected' : 'disconnected'
  });
});

// Start server
app.listen(PORT, '0.0.0.0', async () => {
  console.log(`Backend API listening on port ${PORT}`);
  console.log(`Pod: ${os.hostname()}`);
  console.log(`Cassandra Host: ${process.env.CASSANDRA_HOST || 'cassandra.cassandra.svc.cluster.local'}`);
  
  // Initialize Cassandra after server starts
  await initializeCassandra();
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('Received SIGTERM, shutting down gracefully');
  await cassandraClient.shutdown();
  process.exit(0);
});

process.on('SIGINT', async () => {
  console.log('Received SIGINT, shutting down gracefully');
  await cassandraClient.shutdown();
  process.exit(0);
});
