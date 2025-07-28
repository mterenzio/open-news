# Development Setup Guide

## Prerequisites

### Option 1: PostgreSQL with Homebrew (Recommended for macOS)

1. **Install PostgreSQL**:
   ```bash
   # Install PostgreSQL
   brew install postgresql@15
   
   # Start PostgreSQL service
   brew services start postgresql@15
   
   # Create database
   createdb open_news
   ```

2. **Configure environment**:
   ```bash
   cp .env.example .env
   ```
   
   Your `.env` should look like this for local PostgreSQL:
   ```
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=$(whoami)     # Your macOS username
   DB_PASSWORD=          # Leave empty for local setup
   DB_NAME=open_news
   DB_SSLMODE=disable
   ```

### Option 2: Docker (if Docker is installed)

1. **Install Docker Desktop** from https://docker.com if not already installed

2. **Start PostgreSQL container**:
   ```bash
   docker run --name postgres-opennews \
     -e POSTGRES_PASSWORD=password \
     -e POSTGRES_DB=open_news \
     -p 5432:5432 \
     -d postgres:15
   ```

3. **Configure environment**:
   ```bash
   cp .env.example .env
   ```
   
   Your `.env` should look like this for Docker:
   ```
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=password
   DB_NAME=open_news
   DB_SSLMODE=disable
   ```

## Quick Start with Docker

For the fastest setup, use Docker Compose:

```bash
# Start PostgreSQL and pgAdmin
docker-compose up postgres pgadmin -d

# Wait a few seconds for PostgreSQL to start, then run the app
make run
```

## Manual Setup

1. **Install dependencies**:
   ```bash
   make deps
   ```

2. **Start PostgreSQL** (if not using Docker)

3. **Run database migrations and seed**:
   ```bash
   make run  # This will run migrations automatically
   # In another terminal:
   make seed  # Seeds with popular sources
   
   # Or seed with a custom Bluesky user:
   ./dev seed -handle your.handle.bsky.social
   ```

4. **Test the application**:
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/api/worker/status
   curl http://localhost:8080/api/feeds/global
   ```

## Database Seeding

The application provides flexible seeding options for development and testing:

### Complete Database Reset and Seed (Recommended for Development)
```bash
./dev reset && go run cmd/seed.go -handle your.handle.bsky.social
```
**Complete fresh start**: Drops all tables, runs migrations, and seeds with real Bluesky user data. This workflow:
- Resets the entire database to a clean state
- Runs all migrations to recreate schema
- Creates a user record with actual profile data from Bluesky
- Imports their real follows as sources (authenticates with Bluesky)
- Seeds with mock articles for testing

This is the recommended workflow for consistent development environments.

### Popular Sources (Default)
```bash
make seed
```
Seeds the database with popular news sources and tech personalities from Bluesky.

### Custom Bluesky User
```bash
./dev seed -handle your.handle.bsky.social
```
Creates a test user based on a real Bluesky account. This will:
- Create a user record with the actual profile data
- Import their followers as sources 
- Seed articles from their network

### Custom DID
```bash
./dev seed -did did:plc:your-did-here
```
Seeds using a specific Bluesky DID if you know it.

### Command Options
- `-handle`: Specify a Bluesky handle (e.g., `jay.bsky.team`)
- `-did`: Specify a Bluesky DID directly
- No flags: Uses default popular sources

## Development Workflow

1. **Run with live reload** (install air first):
   ```bash
   make install-air
   make dev
   ```

2. **Check worker status**:
   ```bash
   curl http://localhost:8080/api/worker/status
   ```

3. **View logs**: The application logs will show firehose connections and article discoveries

## Troubleshooting

### Database Connection Issues

- Make sure PostgreSQL is running: `pg_isready`
- Check your `.env` file settings
- For Docker: `docker logs postgres-opennews`

### Firehose Connection Issues

- The firehose connection may take a few attempts to establish
- Check logs for connection errors
- Ensure internet connectivity

### No Articles Appearing

- First seed some sources: `make seed` (popular sources) or `./dev seed -handle your.handle.bsky.social` (custom user)
- The firehose only shows articles from sources in your database
- It may take time for articles appear depending on posting activity

## Testing

For comprehensive testing procedures and commands, see [TESTING.md](TESTING.md).

Quick testing checklist:
1. Visit http://localhost:8080 for the developer dashboard
2. Test health endpoint: http://localhost:8080/health
3. Check worker status: http://localhost:8080/api/worker/status
4. Test feeds: http://localhost:8080/api/feeds/global

The web interface at http://localhost:8080 provides copy-paste commands and live API links for easy testing.
