const express = require('express');
const axios = require('axios');
const os = require('os');
const path = require('path');
const app = express();
const PORT = 8080;

const BACKEND_URL = process.env.BACKEND_URL || 'http://backend-service:5000';

// Serve static files
app.use(express.static(path.join(__dirname, 'public')));
app.use(express.json());

// API route to get job data from backend
app.get('/api/job', async (req, res) => {
  try {
    const response = await axios.get(`${BACKEND_URL}/`);
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching job from backend:', error.message);
    res.status(500).json({ 
      error: 'Failed to connect to backend service',
      details: error.message 
    });
  }
});

// API route to get all jobs
app.get('/api/jobs', async (req, res) => {
  try {
    const response = await axios.get(`${BACKEND_URL}/jobs`);
    res.json(response.data);
  } catch (error) {
    console.error('Error fetching jobs from backend:', error.message);
    res.status(500).json({ 
      error: 'Failed to fetch jobs from backend',
      details: error.message 
    });
  }
});

// API route to create new job
app.post('/api/jobs', async (req, res) => {
  try {
    const response = await axios.post(`${BACKEND_URL}/jobs`, req.body);
    res.json(response.data);
  } catch (error) {
    console.error('Error creating job:', error.message);
    res.status(500).json({ 
      error: 'Failed to create job',
      details: error.message 
    });
  }
});

// API route to check backend health
app.get('/api/health', async (req, res) => {
  try {
    const response = await axios.get(`${BACKEND_URL}/health`);
    res.json(response.data);
  } catch (error) {
    console.error('Backend health check failed:', error.message);
    res.status(500).json({ 
      error: 'Backend service unavailable',
      details: error.message 
    });
  }
});

// Main route - serve the frontend application
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'public', 'index.html'));
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`Frontend server listening on port ${PORT}`);
  console.log(`Pod: ${os.hostname()}`);
  console.log(`Backend URL: ${BACKEND_URL}`);
});