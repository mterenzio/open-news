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

## Admin Interface

The application includes an admin interface for managing articles, users, and sources. Access it at http://localhost:8080/admin/ (password: `admin123`).

### Article Management

The admin interface provides several tools for managing articles:

1. **Article Inspection**: View detailed information about any article including:
   - Basic metadata (title, author, publication date)
   - JSON-LD structured data (for NewsArticle validation)
   - Open Graph metadata
   - Content statistics and quality metrics

2. **URL Validation Testing**: Test if any URL would be accepted as a valid NewsArticle:
   ```bash
   # Test if a URL has valid NewsArticle JSON-LD schema
   curl -s -u admin:admin123 "http://localhost:8080/admin/inspect?url=https://example.com/article" | jq .
   
   # Example with a real TechCrunch article:
   curl -s -u admin:admin123 "http://localhost:8080/admin/inspect?url=https://techcrunch.com/2025/07/28/microsoft-edge-is-now-an-ai-browser-with-launch-of-copilot-mode/" | jq .
   ```
   This endpoint returns:
   - `isNewsArticle`: boolean indicating if the URL contains valid NewsArticle schema
   - `error`: any validation or network errors encountered
   - Useful for debugging why articles from specific sources aren't being captured
   
   **Note**: The validation supports both simple JSON-LD structures and complex `@graph` arrays commonly used by major news sites.

3. **Article Validation and Cleanup**: Remove articles that don't meet NewsArticle schema requirements
   ```bash
   # Dry run (preview what would be deleted)
   curl -X POST -u admin:admin123 "http://localhost:8080/admin/validate-articles?dry_run=true"
   
   # Actually delete invalid articles
   curl -X POST -u admin:admin123 "http://localhost:8080/admin/validate-articles?dry_run=false"
   ```

3. **Real-time Validation**: The system now validates all articles at ingestion time:
   - Firehose processing validates NewsArticle schema before storing articles
   - Manual article imports through ArticlesService also validate schema
   - Only articles with proper JSON-LD NewsArticle markup are stored

### Admin Features

- **Users Page**: View and manage user accounts and follows
- **Sources Page**: Manage news sources and their quality scores  
- **Articles Page**: Browse all articles with filtering and search
- **Individual Article Inspection**: Deep dive into article metadata and validation status
- **Follow Management**: Refresh user follows from Bluesky API

### Data Quality Assurance

The application enforces strict data quality standards:

- **NewsArticle Validation**: All articles must have valid JSON-LD NewsArticle schema
- **Automatic Cleanup**: Invalid articles are prevented from entering the database
- **Manual Cleanup**: Admin can run validation to clean up historical invalid data
- **Content Validation**: Articles are validated for title, description, and publication metadata

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
