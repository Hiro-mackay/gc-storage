# GC Storage

A full-stack cloud storage system with file management, sharing, and team collaboration features.

---

## Overview

GC Storage is a cloud storage platform built from scratch, designed for both personal use and team collaboration. It provides secure file storage, sharing, and management with a modern tech stack.

### Features

- **File Management** - Upload, download, preview, and organize files with folder hierarchy
- **Version Control** - Track file changes with automatic versioning and restore capabilities
- **Team Collaboration** - Create groups, manage members, and share resources
- **Flexible Sharing** - Generate share links with password protection and expiration
- **Fine-grained Permissions** - PBAC + ReBAC authorization model
- **Full-text Search** - Search files by name, type, date, and metadata

### Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Go 1.22+ / Echo v4 / Clean Architecture |
| **Frontend** | React 19 / TanStack Router & Query / Zustand / Tailwind CSS |
| **Database** | PostgreSQL 16 / Redis 7 |
| **Storage** | MinIO (S3-compatible) |
| **Auth** | JWT + OAuth 2.0 (Google, GitHub) |

---

## Quick Start

### Prerequisites

- [Go](https://go.dev/) 1.22+
- [Node.js](https://nodejs.org/) 20+
- [pnpm](https://pnpm.io/) 9+
- [Docker](https://www.docker.com/) & Docker Compose
- [Task](https://taskfile.dev/) (task runner)

### Installation

```bash
# Clone the repository
git clone https://github.com/Hiro-mackay/gc-storage.git
cd gc-storage

# Check if all required tools are installed
task doctor

# Install development tools and dependencies
task setup
```

### Start Development Environment

```bash
# One command to start everything (infra + backend + frontend)
task dev
```

This will:
1. Start infrastructure (PostgreSQL, Redis, MinIO, MailHog)
2. Wait for all services to be healthy
3. Apply database migrations
4. Start backend server with hot reload (http://localhost:8080)
5. Start frontend dev server (http://localhost:3000)

### Alternative Start Options

```bash
# Start backend only (no frontend)
task dev:backend

# Start frontend only (assumes backend is already running)
task dev:frontend

# Start services in parallel (requires infra already running)
task dev:services
```

### Stop Environment

```bash
# Stop services (keep data)
task infra:down

# Stop and remove all data (WARNING: destructive)
task infra:destroy
```

---

## Project Structure

```
gc-storage/
├── backend/                 # Go API server
│   ├── cmd/api/            # Entry point
│   ├── internal/
│   │   ├── domain/         # Entities, Value Objects, Repository Interfaces
│   │   ├── usecase/        # Business logic (CQRS pattern)
│   │   ├── interface/      # Handlers, Middleware, DTOs
│   │   └── infrastructure/ # Repository implementations
│   │       └── database/
│   │           ├── migrations/  # Database migrations
│   │           ├── queries/     # SQLC query definitions
│   │           └── sqlcgen/     # SQLC generated code
│   ├── tests/integration/  # Integration tests
│   └── pkg/                # Shared packages
├── frontend/               # React SPA
│   └── src/
│       ├── app/routes/     # TanStack Router
│       ├── components/     # UI components
│       ├── features/       # Feature modules
│       ├── stores/         # Zustand stores
│       └── lib/            # Utilities
├── docs/                   # Documentation
│   ├── 01-policies/       # Development policies
│   ├── 02-architecture/   # Technical architecture
│   ├── 03-domains/        # Domain definitions
│   ├── 04-specs/          # Feature specifications
│   └── 05-operations/     # Operations guides
├── docker-compose.yml      # Local infrastructure
├── Taskfile.yml           # Task runner config
└── .env.local             # Local environment variables
```

---

## Development Commands

### Quick Reference

| Command | Description |
|---------|-------------|
| `task` | Show all available tasks |
| `task dev` | Start full development environment |
| `task check` | Run all checks (lint + test) |
| `task doctor` | Check if all required tools are installed |

### Development

```bash
task dev              # Start full environment (infra + backend + frontend)
task dev:backend      # Start infra + backend only
task dev:frontend     # Start frontend only
task dev:services     # Start backend + frontend (infra must be running)
```

### Backend (Go)

```bash
task backend:dev          # Start with hot reload (Air)
task backend:run          # Run without hot reload
task backend:build        # Build binary
task backend:test         # Run unit tests
task backend:test-integration  # Run integration tests
task backend:test-coverage     # Run tests with coverage report
task backend:lint         # Run golangci-lint
task backend:lint-fix     # Run golangci-lint with auto-fix
task backend:fmt          # Format code
task backend:sqlc         # Generate typed SQL
task backend:mocks        # Generate mocks
```

### Frontend (React)

```bash
task frontend:dev         # Start Vite dev server
task frontend:build       # Production build
task frontend:preview     # Preview production build
task frontend:test        # Run tests
task frontend:test-watch  # Run tests in watch mode
task frontend:test-coverage   # Run tests with coverage
task frontend:lint        # Run ESLint
task frontend:fmt         # Format with Prettier
```

### Testing

```bash
task test             # Run all tests (starts infra, runs unit + integration)
task test:unit        # Run unit tests only (no infra needed)
task test:integration # Run integration tests only (starts infra)
task check            # Run lint + test (quick validation)
```

### Database

```bash
task migrate:up           # Apply all pending migrations
task migrate:down         # Rollback last migration
task migrate:reset        # Rollback ALL migrations (dangerous)
task migrate:create NAME=xxx  # Create new migration
task migrate:version      # Show current version
task migrate:force VERSION=1  # Force set version (use with caution)
task db:connect           # Connect via psql
task db:reset             # Reset database (drop + recreate + migrate)
```

### Infrastructure

```bash
task infra:up         # Start Docker services
task infra:down       # Stop services (keep data)
task infra:destroy    # Stop and remove all volumes (deletes data)
task infra:restart    # Restart services
task infra:wait       # Wait for services to be healthy
task infra:logs       # View logs (follow mode)
task infra:status     # Check status
```

### Code Quality

```bash
task lint             # Run all linters
task fmt              # Format all code
task check            # Run lint + test
task ci               # Run CI pipeline (lint + unit test + build)
task ci:full          # Run full CI (includes integration tests)
```

### Setup & Utilities

```bash
task setup            # Install tools + dependencies
task setup:tools      # Install Go development tools
task setup:deps       # Install project dependencies
task setup:backend    # Install backend dependencies
task setup:frontend   # Install frontend dependencies
task doctor           # Check required tools
task clean            # Clean build artifacts
```

### Open in Browser

```bash
task open:frontend    # Open http://localhost:3000
task open:api         # Open http://localhost:8080/api/v1
task open:minio       # Open MinIO console
task open:mailhog     # Open MailHog UI
```

---

## Local Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Frontend | http://localhost:3000 | - |
| Backend API | http://localhost:8080 | - |
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin |
| MailHog | http://localhost:8025 | - |
| PostgreSQL | localhost:5432 | postgres / postgres |
| Redis | localhost:6379 | - |

---

## Documentation

Detailed documentation is available in the `docs/` directory:

- **[Setup Guide](docs/01-policies/SETUP.md)** - Development environment setup
- **[Coding Standards](docs/01-policies/CODING_STANDARDS.md)** - Code conventions
- **[TDD Workflow](docs/01-policies/TDD_WORKFLOW.md)** - Test-driven development process
- **[System Architecture](docs/02-architecture/SYSTEM.md)** - System overview
- **[Backend Architecture](docs/02-architecture/BACKEND.md)** - Backend design
- **[API Design](docs/02-architecture/API.md)** - API specifications
- **[Database Schema](docs/02-architecture/DATABASE.md)** - Schema design

---

## License

MIT
