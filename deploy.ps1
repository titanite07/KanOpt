# KanOpt Deployment Script for Windows PowerShell
# This script handles the complete deployment of the Agentic Kanban Sprint Optimizer

param(
    [Parameter(Position=0)]
    [string]$Command = "deploy",
    [Parameter(Position=1)]
    [string]$Service = "",
    [Parameter(Position=2)]
    [string]$Count = ""
)

# Function to print colored output
function Write-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Check prerequisites
function Test-Prerequisites {
    Write-Status "Checking prerequisites..."
    
    try {
        docker --version | Out-Null
    } catch {
        Write-Error "Docker is not installed or not in PATH. Please install Docker Desktop."
        exit 1
    }
    
    try {
        docker-compose --version | Out-Null
    } catch {
        Write-Error "Docker Compose is not available. Please ensure Docker Desktop is installed."
        exit 1
    }
    
    Write-Success "Prerequisites check passed"
}

# Clean up previous deployment
function Remove-PreviousDeployment {
    Write-Status "Cleaning up previous deployment..."
    try {
        docker-compose down --remove-orphans --volumes 2>$null
        docker system prune -f 2>$null
    } catch {
        # Ignore errors during cleanup
    }
    Write-Success "Cleanup completed"
}

# Build and start services
function Start-Deployment {
    Write-Status "Starting KanOpt deployment..."
    
    # Start infrastructure services first
    Write-Status "Starting infrastructure services..."
    docker-compose up -d postgres redis rabbitmq
    
    # Wait for infrastructure to be ready
    Write-Status "Waiting for infrastructure services to be ready..."
    Start-Sleep -Seconds 30
    
    # Start application services
    Write-Status "Starting application services..."
    docker-compose up -d backend-api ai-service allocator-agent
    
    # Wait for backend services
    Write-Status "Waiting for backend services to be ready..."
    Start-Sleep -Seconds 45
    
    # Start frontend and nginx
    Write-Status "Starting frontend and proxy..."
    docker-compose up -d frontend nginx
    
    # Start monitoring services
    Write-Status "Starting monitoring services..."
    docker-compose up -d prometheus grafana
    
    Write-Success "All services started successfully"
}

# Check service health
function Test-ServiceHealth {
    Write-Status "Checking service health..."
    
    $services = @("postgres", "redis", "rabbitmq", "backend-api", "ai-service", "allocator-agent", "frontend")
    
    foreach ($service in $services) {
        $status = docker-compose ps | Select-String $service
        if ($status -and $status -match "Up") {
            Write-Success "$service is running"
        } else {
            Write-Error "$service is not running"
        }
    }
}

# Display access information
function Show-AccessInfo {
    Write-Success "KanOpt deployment completed successfully!"
    Write-Host ""
    Write-Host "🌐 Access URLs:" -ForegroundColor Cyan
    Write-Host "  Frontend:            http://localhost:3000"
    Write-Host "  API Documentation:   http://localhost:8080/docs"
    Write-Host "  AI Service:          http://localhost:8000/docs"
    Write-Host "  RabbitMQ Management: http://localhost:15672"
    Write-Host "  Grafana Dashboard:   http://localhost:3001"
    Write-Host "  Prometheus:          http://localhost:9090"
    Write-Host ""
    Write-Host "🔐 Default Credentials:" -ForegroundColor Cyan
    Write-Host "  RabbitMQ: kanopt / kanopt123"
    Write-Host "  Database: kanopt / kanopt123"
    Write-Host "  Grafana:  admin / admin"
    Write-Host ""
    Write-Host "📊 Monitoring:" -ForegroundColor Cyan
    Write-Host "  View logs: docker-compose logs -f [service-name]"
    Write-Host "  Check status: docker-compose ps"
    Write-Host "  Stop services: docker-compose down"
    Write-Host ""
    Write-Warning "Note: Services may take a few minutes to fully initialize"
}

# Main script logic
function Main {
    Write-Host "🚀 KanOpt - Agentic Kanban Sprint Optimizer" -ForegroundColor Magenta
    Write-Host "============================================" -ForegroundColor Magenta
    Write-Host ""
    
    switch ($Command) {
        "clean" {
            Remove-PreviousDeployment
        }
        "deploy" {
            Test-Prerequisites
            Start-Deployment
            Start-Sleep -Seconds 10
            Test-ServiceHealth
            Show-AccessInfo
        }
        "health" {
            Test-ServiceHealth
        }
        "logs" {
            if ($Service) {
                docker-compose logs -f $Service
            } else {
                docker-compose logs -f
            }
        }
        "stop" {
            Write-Status "Stopping all services..."
            docker-compose down
            Write-Success "All services stopped"
        }
        "restart" {
            Write-Status "Restarting all services..."
            docker-compose restart
            Write-Success "All services restarted"
        }
        "scale" {
            if ($Service -and $Count) {
                Write-Status "Scaling $Service to $Count instances..."
                docker-compose up -d --scale "$Service=$Count"
                Write-Success "Scaling completed"
            } else {
                Write-Error "Usage: .\deploy.ps1 scale <service> <count>"
                exit 1
            }
        }
        default {
            Write-Host "Usage: .\deploy.ps1 [command] [service] [count]" -ForegroundColor Yellow
            Write-Host ""
            Write-Host "Commands:" -ForegroundColor Yellow
            Write-Host "  deploy   - Full deployment (default)"
            Write-Host "  clean    - Clean up previous deployment"
            Write-Host "  health   - Check service health"
            Write-Host "  logs     - View logs (optional: service name)"
            Write-Host "  stop     - Stop all services"
            Write-Host "  restart  - Restart all services"
            Write-Host "  scale    - Scale a service (usage: scale <service> <count>)"
            Write-Host ""
            Write-Host "Examples:" -ForegroundColor Yellow
            Write-Host "  .\deploy.ps1 deploy"
            Write-Host "  .\deploy.ps1 logs backend-api"
            Write-Host "  .\deploy.ps1 scale backend-api 3"
        }
    }
}

# Run the main function
Main
