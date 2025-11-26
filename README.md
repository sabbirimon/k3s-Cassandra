# Kubernetes Networking Lab with Cassandra Database

This comprehensive Kubernetes networking lab demonstrates service communication, DNS resolution, load balancing, and database integration using a multi-tier application architecture with Apache Cassandra as the backend database.

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │    Backend      │    │   Cassandra     │
│   (Node.js)     │◄──►│    (Node.js)    │◄──►│   Cluster       │
│   Port: 8080    │    │   Port: 5000    │    │   Port: 9042    │
│   Replicas: 3   │    │   Replicas: 3   │    │   Replicas: 6   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
       │                       │                       │
       └───────────────────────┼───────────────────────┘
                               │
                    ┌─────────────────┐
                    │  Kubernetes     │
                    │  Services      │
                    │  (ClusterIP,   │
                    │   NodePort,    │
                    │ LoadBalancer)  │
                    └─────────────────┘
```

## 🚀 Features

### Backend Service
- **Cassandra Integration**: Full CRUD operations with Apache Cassandra
- **RESTful API**: REST endpoints for job management
- **Health Checks**: Liveness and readiness probes
- **Service Discovery**: DNS-based service communication
- **Load Balancing**: Multiple replicas with automatic load distribution

### Frontend Application
- **Modern UI**: Responsive web interface with real-time updates
- **Service Communication**: Demonstrates inter-service networking
- **Job Management**: Create, view, and manage jobs through the UI
- **Health Monitoring**: Real-time service health status
- **Auto-refresh**: Automatic data updates every 30 seconds

### Cassandra Database
- **Multi-Region**: Configured for geographic distribution
- **High Availability**: 6-node cluster with replication
- **Persistent Storage**: 100Gi SSD storage per node
- **Monitoring**: Prometheus metrics and JMX integration
- **Security**: Network policies and RBAC configuration

## 📋 Prerequisites

### System Requirements

#### Kubernetes Cluster
- **Version**: Kubernetes 1.20+ 
- **Nodes**: Minimum 3 worker nodes
- **Memory**: 8GB+ per node
- **CPU**: 4+ cores per node
- **Storage**: 200GB+ total available storage

#### Software Dependencies
```bash
# Required CLI tools
kubectl version 1.20+
docker version 20.10+
helm version 3.0+ (optional)

# Node.js (for local development)
node version 16+
npm version 8+
```

#### Cluster Configuration
- **Container Runtime**: containerd or Docker
- **Network Plugin**: Calico, Flannel, or Weave
- **Ingress Controller**: NGINX or Traefik (for LoadBalancer services)
- **Storage Classes**: SSD storage classes available

## 🛠️ Installation Guide

### Step 1: Clone and Prepare Repository

```bash
# Clone the repository
git clone https://github.com/sabbirimon/k3s-Cassandra.git
cd k3s-Cassandra

# Verify cluster connectivity
kubectl cluster-info
kubectl get nodes
```

### Step 2: Build Application Images

```bash
# Build backend image
cd backend
docker build -t backend-api:latest .
docker tag backend-api:latest your-registry/backend-api:latest
docker push your-registry/backend-api:latest

# Build frontend image
cd ../frontend
docker build -t frontend-app:latest .
docker tag frontend-app:latest your-registry/frontend-app:latest
docker push your-registry/frontend-app:latest

# Update image references in YAML files if needed
sed -i 's/backend-api:latest/your-registry\/backend-api:latest/g' ../backend-deployment.yaml
sed -i 's/frontend-app:latest/your-registry\/frontend-app:latest/g' ../frontend-service.yaml
```

### Step 3: Deploy Infrastructure Components

```bash
# 1. Create namespace and RBAC
kubectl apply -f namespace-rbac.yaml

# 2. Create storage classes
kubectl apply -f storage-classes.yaml

# 3. Create Cassandra configuration
kubectl apply -f cassandra-config.yaml

# 4. Deploy security policies
kubectl apply -f security-policies.yaml

# 5. Deploy Cassandra cluster
kubectl apply -f cassandra-statefulset.yaml

# 6. Deploy monitoring
kubectl apply -f monitoring.yaml
```

### Step 4: Deploy Application Services

```bash
# Deploy backend service
kubectl apply -f backend-service.yaml

# Deploy frontend service
kubectl apply -f frontend-service.yaml

# Optional: Deploy LoadBalancer service
kubectl apply -f frontend-service-lb.yaml
```

### Step 5: Verify Deployment

```bash
# Check all pods
kubectl get pods -o wide --all-namespaces

# Check services
kubectl get svc --all-namespaces

# Check Cassandra cluster status
kubectl exec -it cassandra-0 -n cassandra -- nodetool status

# Check application logs
kubectl logs -l app=backend -f
kubectl logs -l app=frontend -f
```

## 🔧 Configuration

### Environment Variables

#### Backend Configuration
```yaml
env:
- name: CASSANDRA_HOST
  value: "cassandra.cassandra.svc.cluster.local"
- name: CASSANDRA_KEYSPACE
  value: "job_tracker"
- name: CASSANDRA_DC
  value: "datacenter1"
```

#### Frontend Configuration
```yaml
env:
- name: BACKEND_URL
  value: "http://backend-service:5000"
```

### Cassandra Configuration

#### Keyspace Schema
```cql
CREATE KEYSPACE job_tracker 
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 3};

CREATE TABLE jobs (
  id UUID PRIMARY KEY,
  title TEXT,
  description TEXT,
  status TEXT,
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  assigned_to TEXT,
  priority INT
);
```

#### Cluster Settings
- **Replication Factor**: 3 (adjust based on node count)
- **Consistency Level**: QUORUM
- **Compaction Strategy**: SizeTieredCompactionStrategy
- **Snitch**: GossipingPropertyFileSnitch

## 🌐 Accessing the Application

### NodePort Access
```bash
# Get node IP
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="ExternalIP")].address}')

# Access frontend
curl http://$NODE_IP:30080
```

### LoadBalancer Access
```bash
# Get LoadBalancer IP
LB_IP=$(kubectl get svc frontend-service-lb -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Access frontend
curl http://$LB_IP
```

### Port Forwarding (Development)
```bash
# Forward frontend port
kubectl port-forward svc/frontend-service 8080:80

# Forward backend port
kubectl port-forward svc/backend-service 5000:5000

# Access locally
curl http://localhost:8080
```

## 🔍 Testing and Validation

### Health Checks
```bash
# Backend health
curl http://backend-service:5000/health

# Frontend health
curl http://frontend-service/health

# Cassandra connectivity
kubectl exec -it cassandra-0 -n cassandra -- cqlsh -e "DESCRIBE KEYSPACES;"
```

### Load Testing
```bash
# Install Apache Bench
sudo apt-get install apache2-utils

# Test backend API
ab -n 1000 -c 10 http://backend-service:5000/

# Test frontend
ab -n 1000 -c 10 http://frontend-service/
```

### Network Diagnostics
```bash
# Test DNS resolution
kubectl run dns-test --image=busybox --rm -it -- nslookup backend-service

# Test service connectivity
kubectl run curl-test --image=curlimages/curl --rm -it -- curl http://backend-service:5000/health

# Check iptables rules
kubectl run iptables-debug --privileged --image=ubuntu --rm -it -- iptables -t nat -L KUBE-SERVICES
```

## 📊 Monitoring and Observability

### Prometheus Metrics
Cassandra metrics are exposed through the Prometheus exporter:

- **JMX Metrics**: Memory, CPU, disk usage
- **Database Metrics**: Read/write latency, throughput
- **Cluster Metrics**: Node status, replication lag

### Log Aggregation
```bash
# View application logs
kubectl logs -l app=backend --tail=100
kubectl logs -l app=frontend --tail=100

# View Cassandra logs
kubectl logs -l app=cassandra -n cassandra --tail=100
```

### Health Monitoring
```bash
# Check pod status
kubectl get pods -o wide

# Check service endpoints
kubectl get endpoints

# Check resource usage
kubectl top pods
kubectl top nodes
```

## 🔒 Security Configuration

### Network Policies
- **Cassandra Isolation**: Only backend services can access Cassandra
- **Frontend Access**: Frontend can only communicate with backend
- **Ingress Control**: External access controlled through services

### RBAC Configuration
- **Service Accounts**: Dedicated service accounts for each component
- **Role Binding**: Minimal required permissions
- **Security Context**: Non-root containers where possible

### Secrets Management
```bash
# Create secrets for sensitive data
kubectl create secret generic cassandra-secrets \
  --from-literal=username=cassandra \
  --from-literal=password=your-secure-password
```

## 🚨 Troubleshooting

### Common Issues

#### Cassandra Cluster Issues
```bash
# Check cluster status
kubectl exec -it cassandra-0 -n cassandra -- nodetool status

# Check seed node connectivity
kubectl exec -it cassandra-0 -n cassandra -- nodetool gossipinfo

# View logs
kubectl logs cassandra-0 -n cassandra
```

#### Backend Connection Issues
```bash
# Check backend pods
kubectl get pods -l app=backend

# Check service endpoints
kubectl get endpoints backend-service

# Test connectivity
kubectl run test-pod --image=busybox --rm -it -- wget -qO- http://backend-service:5000/health
```

#### Frontend Issues
```bash
# Check frontend pods
kubectl get pods -l app=frontend

# Check service configuration
kubectl describe svc frontend-service

# View browser console for JavaScript errors
```

### Performance Tuning

#### Cassandra Optimization
```yaml
# JVM settings in cassandra-statefulset.yaml
env:
- name: MAX_HEAP_SIZE
  value: "4G"
- name: HEAP_NEWSIZE
  value: "800M"
- name: CASSANDRA_NUM_TOKENS
  value: "256"
```

#### Application Scaling
```bash
# Scale backend
kubectl scale deployment backend-deployment --replicas=5

# Scale frontend
kubectl scale deployment frontend-deployment --replicas=5

# Scale Cassandra (careful with data consistency)
kubectl scale statefulset cassandra --replicas=9 -n cassandra
```

## 🔄 CI/CD Integration

### GitHub Actions Example
```yaml
name: Deploy to Kubernetes
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - name: Build and push Docker images
      run: |
        docker build -t backend-api:${{ github.sha }} ./backend
        docker build -t frontend-app:${{ github.sha }} ./frontend
        # Push to registry
    - name: Deploy to Kubernetes
      run: |
        kubectl set image deployment/backend-deployment backend=backend-api:${{ github.sha }}
        kubectl set image deployment/frontend-deployment frontend=frontend-app:${{ github.sha }}
```

## 📚 Learning Objectives

After completing this lab, you will understand:

1. **Kubernetes Networking**
   - Service types (ClusterIP, NodePort, LoadBalancer)
   - DNS resolution and service discovery
   - Network policies and security

2. **Database Integration**
   - Stateful applications with StatefulSets
   - Persistent volume management
   - Database clustering and replication

3. **Application Architecture**
   - Multi-tier application design
   - Inter-service communication
   - Load balancing and scaling

4. **Operations and Monitoring**
   - Health checks and probes
   - Logging and observability
   - Troubleshooting techniques

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For issues and questions:
- Create an issue in the GitHub repository
- Check the troubleshooting section
- Review Kubernetes documentation

---

**Happy Learning! 🎓**