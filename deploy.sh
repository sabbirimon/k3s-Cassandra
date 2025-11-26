#!/bin/bash

# Kubernetes Networking Lab Deployment Script
# This script deploys the complete Cassandra + Kubernetes networking demo

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
    log_info "Building Docker images..."
    
    # Build backend image
    log_info "Building backend image..."
    docker build -t backend-api:latest ./backend
    if [ $? -eq 0 ]; then
        log_success "Backend image built successfully"
    else
        log_error "Failed to build backend image"
        exit 1
    fi
    
    # Build frontend image
    log_info "Building frontend image..."
    docker build -t frontend-app:latest ./frontend
    if [ $? -eq 0 ]; then
        log_success "Frontend image built successfully"
    else
        log_error "Failed to build frontend image"
        exit 1
    fi
    
    log_success "All images built successfully"
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
    log_info "Deploying application services..."
    
    # Deploy backend service
    log_info "Deploying backend service..."
    kubectl apply -f backend-service.yaml
    
    # Wait for backend to be ready
    kubectl wait --for=condition=available deployment/backend-deployment --timeout=300s
    
    # Deploy frontend service
    log_info "Deploying frontend service..."
    kubectl apply -f frontend-service.yaml
    
    # Wait for frontend to be ready
    kubectl wait --for=condition=available deployment/frontend-deployment --timeout=300s
    
    log_success "Application deployment completed"
}

verify_deployment() {
    log_info "Verifying deployment..."
    
    # Check all pods
    log_info "Checking pod status..."
    kubectl get pods -o wide --all-namespaces
    
    # Check services
    log_info "Checking service status..."
    kubectl get svc --all-namespaces
    
    # Check Cassandra cluster status
    log_info "Checking Cassandra cluster status..."
    kubectl exec -it cassandra-0 -n cassandra -- nodetool status || true
    
    # Test backend health
    log_info "Testing backend health..."
    kubectl run health-test --image=curlimages/curl --rm -i --restart=Never -- \
        curl -f http://backend-service:5000/health || log_warning "Backend health check failed"
    
    # Test frontend connectivity
    log_info "Testing frontend connectivity..."
    kubectl run frontend-test --image=curlimages/curl --rm -i --restart=Never -- \
        curl -f http://frontend-service/ || log_warning "Frontend connectivity test failed"
    
    log_success "Deployment verification completed"
}

show_access_info() {
    log_info "Access Information:"
    
    # Get node IP
    NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="ExternalIP")].address}' 2>/dev/null || \
              kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
    
    if [ -n "$NODE_IP" ]; then
        echo -e "${GREEN}Frontend Application:${NC}"
        echo -e "  NodePort: http://$NODE_IP:30080"
        echo -e "  Service: http://frontend-service"
        echo ""
        echo -e "${GREEN}Backend API:${NC}"
        echo -e "  Service: http://backend-service:5000"
        echo -e "  Health: http://backend-service:5000/health"
        echo ""
        echo -e "${GREEN}Cassandra:${NC}"
        echo -e "  Connect: kubectl exec -it cassandra-0 -n cassandra -- cqlsh"
        echo -e "  Status: kubectl exec -it cassandra-0 -n cassandra -- nodetool status"
    else
        log_warning "Could not determine node IP address"
    fi
}

cleanup() {
    log_warning "Cleaning up deployment..."
    
    # Delete applications
    kubectl delete -f frontend-service.yaml --ignore-not-found=true
    kubectl delete -f backend-service.yaml --ignore-not-found=true
    
    # Delete infrastructure
    kubectl delete -f monitoring.yaml --ignore-not-found=true
    kubectl delete -f cassandra-statefulset.yaml --ignore-not-found=true
    kubectl delete -f security-policies.yaml --ignore-not-found=true
    kubectl delete -f cassandra-config.yaml --ignore-not-found=true
    kubectl delete -f storage-classes.yaml --ignore-not-found=true
    kubectl delete -f namespace-rbac.yaml --ignore-not-found=true
    
    # Wait for cleanup
    kubectl wait --for=delete pod -l app=cassandra -n cassandra --timeout=300s || true
    
    log_success "Cleanup completed"
}

show_help() {
    echo "Kubernetes Networking Lab Deployment Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  deploy     Deploy the complete application stack"
    echo "  build      Build Docker images only"
    echo "  cleanup    Remove all deployed resources"
    echo "  verify     Verify deployment status"
    echo "  help       Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 deploy    # Deploy everything"
    echo "  $0 build     # Build images only"
    echo "  $0 cleanup   # Clean up deployment"
}

# Main script logic
case "${1:-deploy}" in
    "deploy")
        check_prerequisites
        build_images
        deploy_infrastructure
        deploy_applications
        verify_deployment
        show_access_info
        ;;
    "build")
        check_prerequisites
        build_images
        ;;
    "cleanup")
        cleanup
        ;;
    "verify")
        verify_deployment
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac

log_success "Script completed successfully!"