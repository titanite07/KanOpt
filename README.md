# KanOpt - Agentic Kanban Sprint Optimizer

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](https://www.docker.com/)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB.svg)](https://www.python.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.0+-3178C6.svg)](https://www.typescriptlang.org/)

> An intelligent, event-sourced Kanban board with AI-powered predictive analytics and autonomous task optimization.

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Backend       │    │   AI/ML         │
│   (Next.js)     │    │   (Golang)      │    │   (Python)      │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ • Kanban Board  │◄──►│ • State Manager │◄──►│ • LSTM Predictor│
│ • Drag & Drop   │    │ • Event Store   │    │ • Risk Analysis │
│ • Virtual Scroll│    │ • Allocator     │    │ • Weekly Retrain│
│ • Risk Heatmap  │    │ • Agent REST    │    │ • Velocity Pred │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   RabbitMQ      │
                    │   Event Bus     │
                    │ • Task Events   │
                    │ • Risk Alerts   │
                    │ • Reallocations │
                    └─────────────────┘
```

## 🚀 Core Features

### Event-Sourced Architecture
- **Immutable Event Log**: All drag/drop and status changes stored in RabbitMQ
- **Event Replay**: Complete state reconstruction from event stream
- **Real-time Updates**: Live synchronization across all clients

### High-Performance UI
- **Virtual Scrolling**: Handle thousands of cards with lazy loading
- **Optimized Rendering**: Intersection Observer + requestAnimationFrame + throttling
- **Drag & Drop**: Native DragEvent API with collision detection

### Predictive AI
- **LSTM Velocity Predictor**: Python-based neural network analyzing task movement patterns
- **Weekly Retraining**: Continuous learning from new data
- **Risk Heatmap**: Visual overlay showing prediction confidence

### Autonomous Agent
- **Allocator Agent**: Golang microservice for intelligent task reassignment
- **Risk Response**: Automatically rebalances workload based on AI predictions
- **Human Override**: Users can approve or reject agent suggestions

## 🛠️ Technology Stack

- **Frontend**: Next.js 14, TypeScript, Tailwind CSS, Framer Motion
- **Backend**: Golang, Gin, GORM, Event Sourcing
- **AI/ML**: Python, PyTorch, scikit-learn, FastAPI
- **Message Queue**: RabbitMQ with management UI
- **Database**: PostgreSQL for projections, Redis for caching
- **DevOps**: Docker Compose, hot reload, health checks

## 🏃‍♂️ Quick Start

```bash
# Clone and setup
git clone <repo-url>
cd KanOpt

# Start all services
docker-compose up -d

# Install dependencies
cd frontend && npm install
cd ../backend && go mod tidy
cd ../ai-service && pip install -r requirements.txt

# Development mode
npm run dev:all
```

## 📊 Services

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3000 | Next.js Kanban UI |
| Backend API | 8080 | Golang REST API |
| Allocator Agent | 8081 | Autonomous task reallocation |
| AI Service | 8000 | Python ML predictions |
| RabbitMQ | 5672 | Message queue |
| RabbitMQ UI | 15672 | Management interface |
| PostgreSQL | 5432 | Event store & projections |
| Redis | 6379 | Cache & sessions |

## 🎯 Usage

1. **Create Board**: Set up project with columns and team members
2. **Add Tasks**: Create cards with estimates, priorities, and assignments
3. **Drag & Drop**: Move tasks between columns (events logged automatically)
4. **Monitor Predictions**: Watch AI risk heatmap for bottlenecks
5. **Agent Suggestions**: Review and approve autonomous reallocations
6. **Analytics**: View velocity trends and prediction accuracy

## 🔧 Development

```bash
# Frontend development
cd frontend
npm run dev

# Backend development
cd backend
go run main.go

# AI service development
cd ai-service
python -m uvicorn main:app --reload

# Database migrations
cd backend
go run migrate.go
```

## 📈 Event Types

- `TaskCreated`: New task added
- `TaskMoved`: Drag/drop between columns
- `TaskUpdated`: Metadata changes
- `TaskAssigned`: User assignment changes
- `RiskAlert`: AI prediction above threshold
- `ReallocationSuggested`: Agent recommendation
- `ReallocationApproved/Rejected`: User decision

## 🧠 AI Features

- **Velocity Prediction**: LSTM network predicting completion times
- **Bottleneck Detection**: Identify workflow constraints
- **Load Balancing**: Optimal task distribution recommendations
- **Trend Analysis**: Historical performance insights

Built with ❤️ for agile teams seeking AI-powered optimization.
