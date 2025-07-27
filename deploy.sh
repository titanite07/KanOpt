#!/bin/bash

# KanOpt Deployment Script
# This script handles the complete deployment of the Agentic Kanban Sprint Optimizer

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Clean up previous deployment
cleanup() {
    print_status "Cleaning up previous deployment..."
    docker-compose down --remove-orphans --volumes || true
    docker system prune -f || true
    print_success "Cleanup completed"
}

# Build and start services
deploy() {
    print_status "Starting KanOpt deployment..."
    
    # Start infrastructure services first
    print_status "Starting infrastructure services..."
    docker-compose up -d postgres redis rabbitmq
    
    # Wait for infrastructure to be ready
    print_status "Waiting for infrastructure services to be ready..."
    sleep 30
    
    # Start application services
    print_status "Starting application services..."
    docker-compose up -d backend-api ai-service allocator-agent
    
    # Wait for backend services
    print_status "Waiting for backend services to be ready..."
    sleep 45
    
    # Start frontend and nginx
    print_status "Starting frontend and proxy..."
    docker-compose up -d frontend nginx
    
    # Start monitoring services
    print_status "Starting monitoring services..."
    docker-compose up -d prometheus grafana
    
    print_success "All services started successfully"
}

# Check service health
check_health() {
    print_status "Checking service health..."
    
    services=("postgres" "redis" "rabbitmq" "backend-api" "ai-service" "allocator-agent" "frontend")
    
    for service in "${services[@]}"; do
        if docker-compose ps | grep -q "${service}.*Up"; then
            print_success "${service} is running"
        else
            print_error "${service} is not running"
        fi
    done
}

# Display access information
show_access_info() {
    print_success "KanOpt deployment completed successfully!"
    echo ""
    echo "🌐 Access URLs:"
    echo "  Frontend:            http://localhost:3000"
    echo "  API Documentation:   http://localhost:8080/docs"
    echo "  AI Service:          http://localhost:8000/docs"
    echo "  RabbitMQ Management: http://localhost:15672"
    echo "  Grafana Dashboard:   http://localhost:3001"
    echo "  Prometheus:          http://localhost:9090"
    echo ""
    echo "🔐 Default Credentials:"
    echo "  RabbitMQ: kanopt / kanopt123"
    echo "  Database: kanopt / kanopt123"
    echo "  Grafana:  admin / admin"
    echo ""
    echo "📊 Monitoring:"
    echo "  View logs: docker-compose logs -f [service-name]"
    echo "  Check status: docker-compose ps"
    echo "  Stop services: docker-compose down"
    echo ""
    print_warning "Note: Services may take a few minutes to fully initialize"
}

# Main deployment flow
main() {
    echo "🚀 KanOpt - Agentic Kanban Sprint Optimizer"
    echo "============================================"
    echo ""
    
    case "${1:-deploy}" in
        "clean")
            cleanup
            ;;
        "deploy")
            check_prerequisites
            deploy
            sleep 10
            check_health
            show_access_info
            ;;
        "health")
            check_health
            ;;
        "logs")
            if [ -n "$2" ]; then
                docker-compose logs -f "$2"
            else
                docker-compose logs -f
            fi
            ;;
        "stop")
            print_status "Stopping all services..."
            docker-compose down
            print_success "All services stopped"
            ;;
        "restart")
            print_status "Restarting all services..."
            docker-compose restart
            print_success "All services restarted"
            ;;
        "scale")
            if [ -n "$2" ] && [ -n "$3" ]; then
                print_status "Scaling $2 to $3 instances..."
                docker-compose up -d --scale "$2=$3"
                print_success "Scaling completed"
            else
                print_error "Usage: $0 scale <service> <count>"
                exit 1
            fi
            ;;
        *)
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  deploy   - Full deployment (default)"
            echo "  clean    - Clean up previous deployment"
            echo "  health   - Check service health"
            echo "  logs     - View logs (optional: service name)"
            echo "  stop     - Stop all services"
            echo "  restart  - Restart all services"
            echo "  scale    - Scale a service (usage: scale <service> <count>)"
            echo ""
            echo "Examples:"
            echo "  $0 deploy"
            echo "  $0 logs backend-api"
            echo "  $0 scale backend-api 3"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
