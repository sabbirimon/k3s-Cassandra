#!/bin/bash

# Multi-Language Kubernetes Networking Lab Deployment Script
# This script allows deployment of different language stacks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="cassandra"
BACKEND_REPLICAS=3
FRONTEND_REPLICAS=3
CASSANDRA_REPLICAS=6

# Language options
LANGUAGES=("nodejs" "python" "go")
SELECTED_LANG="nodejs"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    echo "Multi-Language Kubernetes Networking Lab Deployment Script"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  deploy     Deploy complete application stack"
    echo "  build      Build Docker images for all languages"
    echo "  cleanup    Remove all deployed resources"
    echo "  verify     Verify deployment status"
    echo "  help       Show this help message"
    echo ""
    echo "Language Options:"
    echo "  --lang nodejs    Deploy Node.js stack (default)"
    echo "  --lang python    Deploy Python stack"
    echo "  --lang go        Deploy Go stack"
    echo "  --lang all       Deploy all language stacks"
    echo ""
    echo "Examples:"
    echo "  $0 deploy --lang python    # Deploy Python stack only"
    echo "  $0 deploy --lang all       # Deploy all language stacks"
    echo "  $0 build --lang go         # Build Go images only"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    # Check node resources
    NODE_COUNT=$(kubectl get nodes --no-headers | wc -l)
    if [ "$NODE_COUNT" -lt 3 ]; then
        log_warning "Recommended minimum 3 nodes, found $NODE_COUNT"
    fi
    
    log_success "Prerequisites check completed"
}

build_images() {
    local lang=$1
    log_info "Building Docker images for $lang..."
    
    case $lang in
        "nodejs")
            log_info "Building Node.js backend image..."
            docker build -t backend-api:latest ./backend
            log_info "Building Node.js frontend image..."
            docker build -t frontend-app:latest ./frontend
            ;;
        "python")
            log_info "Building Python backend image..."
            docker build -t backend-python:latest ./backend-python
            log_info "Building Python frontend image..."
            docker build -t frontend-python:latest ./frontend-python
            ;;
        "go")
            log_info "Building Go backend image..."
            docker build -t backend-go:latest ./backend-go
            log_info "Building Go frontend image..."
            docker build -t frontend-go:latest ./frontend-go
            ;;
        "all")
            for lang in "${LANGUAGES[@]}"; do
                build_images $lang
            done
            return
            ;;
        *)
            log_error "Unknown language: $lang"
            exit 1
            ;;
    esac
    
    if [ $? -eq 0 ]; then
        log_success "$lang images built successfully"
    else
        log_error "Failed to build $lang images"
        exit 1
    fi
}

deploy_infrastructure() {
    log_info "Deploying infrastructure components..."
    
    # 1. Create namespace and RBAC
    log_info "Creating namespace and RBAC..."
    kubectl apply -f namespace-rbac.yaml
    kubectl wait --for=condition=complete job/cassandra-setup -n cassandra --timeout=300s || true
    
    # 2. Create storage classes
    log_info "Creating storage classes..."
    kubectl apply -f storage-classes.yaml
    
    # 3. Create Cassandra configuration
    log_info "Creating Cassandra configuration..."
    kubectl apply -f cassandra-config.yaml
    
    # 4. Deploy security policies
    log_info "Deploying security policies..."
    kubectl apply -f security-policies.yaml
    
    # 5. Deploy Cassandra cluster
    log_info "Deploying Cassandra cluster..."
    kubectl apply -f cassandra-statefulset.yaml
    
    # Wait for Cassandra to be ready
    log_info "Waiting for Cassandra cluster to be ready..."
    kubectl wait --for=condition=ready pod -l app=cassandra -n cassandra --timeout=600s
    
    # 6. Deploy monitoring
    log_info "Deploying monitoring..."
    kubectl apply -f monitoring.yaml
    
    log_success "Infrastructure deployment completed"
}

deploy_applications() {
    local lang=$1
    log_info "Deploying $lang application services..."
    
    case $lang in
        "nodejs")
            log_info "Deploying Node.js backend service..."
            kubectl apply -f backend-service.yaml
            kubectl wait --for=condition=available deployment/backend-deployment --timeout=300s
            
            log_info "Deploying Node.js frontend service..."
            kubectl apply -f frontend-service.yaml
            kubectl wait --for=condition=available deployment/frontend-deployment --timeout=300s
            ;;
        "python")
            log_info "Deploying Python backend service..."
            kubectl apply -f backend-service-python.yaml
            kubectl wait --for=condition=available deployment/backend-deployment-python --timeout=300s
            
            log_info "Deploying Python frontend service..."
            kubectl apply -f frontend-service-python.yaml
            kubectl wait --for=condition=available deployment/frontend-deployment-python --timeout=300s
            ;;
        "go")
            log_info "Deploying Go backend service..."
            kubectl apply -f backend-service-go.yaml
            kubectl wait --for=condition=available deployment/backend-deployment-go --timeout=300s
            
            log_info "Deploying Go frontend service..."
            kubectl apply -f frontend-service-go.yaml
            kubectl wait --for=condition=available deployment/frontend-deployment-go --timeout=300s
            ;;
        "all")
            for lang in "${LANGUAGES[@]}"; do
                deploy_applications $lang
            done
            return
            ;;
        *)
            log_error "Unknown language: $lang"
            exit 1
            ;;
    esac
    
    log_success "$lang application deployment completed"
}

create_language_specific_manifests() {
    local lang=$1
    
    # Create backend service manifest for specific language
    cat > backend-service-$lang.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: backend-service-$lang
  labels:
    app: backend-$lang
spec:
  type: ClusterIP
  selector:
    app: backend-$lang
  ports:
  - name: http
    protocol: TCP
    port: 5000
    targetPort: 5000
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-deployment-$lang
  labels:
    app: backend-$lang
spec:
  replicas: 3
  selector:
    matchLabels:
      app: backend-$lang
  template:
    metadata:
      labels:
        app: backend-$lang
        language: $lang
    spec:
      containers:
      - name: backend
        image: backend-$lang:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 5000
          name: http
        env:
        - name: CASSANDRA_HOST
          value: "cassandra.cassandra.svc.cluster.local"
        - name: CASSANDRA_KEYSPACE
          value: "job_tracker"
        - name: CASSANDRA_DC
          value: "datacenter1"
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 5000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 5000
          initialDelaySeconds: 5
          periodSeconds: 5
EOF

    # Create frontend service manifest for specific language
    cat > frontend-service-$lang.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: frontend-service-$lang
  labels:
    app: frontend-$lang
spec:
  type: NodePort
  selector:
    app: frontend-$lang
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
    nodePort: $((30080 + $(echo ${LANGUAGES[@]} | grep -o -n $lang | cut -d: -f1) - 1))
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend-deployment-$lang
  labels:
    app: frontend-$lang
spec:
  replicas: 3
  selector:
    matchLabels:
      app: frontend-$lang
  template:
    metadata:
      labels:
        app: frontend-$lang
        language: $lang
    spec:
      containers:
      - name: frontend
        image: frontend-$lang:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: BACKEND_URL
          value: "http://backend-service-$lang:5000"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
EOF
}

verify_deployment() {
    local lang=$1
    log_info "Verifying $lang deployment..."
    
    # Check all pods
    log_info "Checking pod status..."
    kubectl get pods -l language=$lang -o wide
    
    # Check services
    log_info "Checking service status..."
    kubectl get svc -l app=frontend-$lang
    
    # Check Cassandra cluster status
    log_info "Checking Cassandra cluster status..."
    kubectl exec -it cassandra-0 -n cassandra -- nodetool status || true
    
    # Test backend health
    log_info "Testing $lang backend health..."
    kubectl run health-test-$lang --image=curlimages/curl --rm -i --restart=Never -- \
        curl -f http://backend-service-$lang:5000/health || log_warning "$lang backend health check failed"
    
    # Test frontend connectivity
    log_info "Testing $lang frontend connectivity..."
    kubectl run frontend-test-$lang --image=curlimages/curl --rm -i --restart=Never -- \
        curl -f http://frontend-service-$lang/ || log_warning "$lang frontend connectivity test failed"
    
    log_success "$lang deployment verification completed"
}

show_access_info() {
    local lang=$1
    log_info "Access Information for $lang:"
    
    # Get node IP
    NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="ExternalIP")].address}' 2>/dev/null || \
              kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
    
    if [ -n "$NODE_IP" ]; then
        NODE_PORT=$((30080 + $(echo ${LANGUAGES[@]} | grep -o -n $lang | cut -d: -f1) - 1))
        echo -e "${GREEN}$lang Frontend Application:${NC}"
        echo -e "  NodePort: http://$NODE_IP:$NODE_PORT"
        echo -e "  Service: http://frontend-service-$lang"
        echo ""
        echo -e "${GREEN}$lang Backend API:${NC}"
        echo -e "  Service: http://backend-service-$lang:5000"
        echo -e "  Health: http://backend-service-$lang:5000/health"
    else
        log_warning "Could not determine node IP address"
    fi
}

cleanup() {
    local lang=$1
    log_warning "Cleaning up $lang deployment..."
    
    # Delete applications
    kubectl delete -f frontend-service-$lang.yaml --ignore-not-found=true
    kubectl delete -f backend-service-$lang.yaml --ignore-not-found=true
    
    # Wait for cleanup
    kubectl wait --for=delete pod -l language=$lang --timeout=300s || true
    
    log_success "$lang cleanup completed"
}

cleanup_all() {
    log_warning "Cleaning up all deployments..."
    
    # Delete all language-specific deployments
    for lang in "${LANGUAGES[@]}"; do
        cleanup $lang
    done
    
    # Delete infrastructure
    kubectl delete -f monitoring.yaml --ignore-not-found=true
    kubectl delete -f cassandra-statefulset.yaml --ignore-not-found=true
    kubectl delete -f security-policies.yaml --ignore-not-found=true
    kubectl delete -f cassandra-config.yaml --ignore-not-found=true
    kubectl delete -f storage-classes.yaml --ignore-not-found=true
    kubectl delete -f namespace-rbac.yaml --ignore-not-found=true
    
    # Wait for cleanup
    kubectl wait --for=delete pod -l app=cassandra -n cassandra --timeout=300s || true
    
    log_success "All cleanup completed"
}

# Parse command line arguments
COMMAND=${1:-deploy}
LANGUAGE="nodejs"

# Parse language option
for arg in "$@"; do
    case $arg in
        --lang=*)
            LANGUAGE="${arg#*=}"
            ;;
        --lang)
            LANGUAGE="$2"
            shift
            ;;
    esac
done

# Validate language
if [[ ! " ${LANGUAGES[@]} " =~ " ${LANGUAGE} " ]] && [ "$LANGUAGE" != "all" ]; then
    log_error "Invalid language: $LANGUAGE"
    log_info "Valid languages: ${LANGUAGES[*]}, all"
    exit 1
fi

# Main script logic
case $COMMAND in
    "deploy")
        check_prerequisites
        build_images $LANGUAGE
        deploy_infrastructure
        
        if [ "$LANGUAGE" = "all" ]; then
            for lang in "${LANGUAGES[@]}"; do
                create_language_specific_manifests $lang
                deploy_applications $lang
                verify_deployment $lang
                show_access_info $lang
            done
        else
            create_language_specific_manifests $LANGUAGE
            deploy_applications $LANGUAGE
            verify_deployment $LANGUAGE
            show_access_info $LANGUAGE
        fi
        ;;
    "build")
        check_prerequisites
        build_images $LANGUAGE
        ;;
    "cleanup")
        if [ "$LANGUAGE" = "all" ]; then
            cleanup_all
        else
            cleanup $LANGUAGE
        fi
        ;;
    "verify")
        if [ "$LANGUAGE" = "all" ]; then
            for lang in "${LANGUAGES[@]}"; do
                verify_deployment $lang
            done
        else
            verify_deployment $LANGUAGE
        fi
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        log_error "Unknown command: $COMMAND"
        show_help
        exit 1
        ;;
esac

log_success "Script completed successfully!"