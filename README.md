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

# Install development tools
task setup:tools

# Install dependencies
task setup
```

### Start Development Environment

```bash
# One command to start everything
task dev
```

This will:
1. Start infrastructure (PostgreSQL, Redis, MinIO, MailHog)
2. Apply database migrations
3. Start backend server (http://localhost:8080)
4. Start frontend dev server (http://localhost:3000)

### Stop Environment

```bash
task infra:down
```

---

## Project Structure

```
gc-storage/
├── backend/                 # Go API server
│   ├── cmd/api/            # Entry point
│   ├── internal/
│   │   ├── domain/         # Entities, Value Objects, Repository Interfaces
│   │   ├── usecase/        # Business logic
│   │   ├── interface/      # Handlers, Middleware, DTOs
│   │   └── infrastructure/ # Repository implementations
│   └── pkg/                # Shared packages
├── frontend/               # React SPA
│   └── src/
│       ├── app/routes/     # TanStack Router
│       ├── components/     # UI components
│       ├── features/       # Feature modules
│       ├── stores/         # Zustand stores
│       └── lib/            # Utilities
├── migrations/             # Database migrations
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

### Full Stack

```bash
task dev              # Start full development environment
task setup            # Install all dependencies
task clean            # Clean build artifacts
```

### Backend (Go)

```bash
task backend:dev      # Start with hot reload (Air)
task backend:test     # Run tests
task backend:lint     # Run golangci-lint
task backend:build    # Build binary
task backend:sqlc     # Generate typed SQL
```

### Frontend (React)

```bash
task frontend:dev     # Start Vite dev server
task frontend:test    # Run tests
task frontend:lint    # Run ESLint
task frontend:build   # Production build
```

### Database

```bash
task migrate:up       # Apply migrations
task migrate:down     # Rollback last migration
task migrate:create NAME=xxx  # Create new migration
task db:connect       # Connect to PostgreSQL
task db:reset         # Reset database
```

### Infrastructure

```bash
task infra:up         # Start Docker services
task infra:down       # Stop and remove volumes
task infra:logs       # View logs
task infra:status     # Check status
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
- **[System Architecture](docs/02-architecture/SYSTEM.md)** - System overview
- **[API Design](docs/02-architecture/API.md)** - API specifications
- **[Database Schema](docs/02-architecture/DATABASE.md)** - Schema design
- **[Contributing](docs/01-policies/CONTRIBUTING.md)** - Contribution guidelines
