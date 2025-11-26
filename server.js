const express = require('express');
const os = require('os');
const app = express();
const PORT = 5000;

const jobs = [
  'Job 1',
  'Job 2',
  'Job 3',
  'Job 4',
  'Job 5',
];

app.get('/', (req, res) => {
  const randomJob = jobs[Math.floor(Math.random() * jobs.length)];
  res.json({
    job: randomJob,
    pod: os.hostname(),
    podIP: req.connection.localAddress
  });
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`Backend API listening on port ${PORT}`);
  console.log(`Pod: ${os.hostname()}`);
});
