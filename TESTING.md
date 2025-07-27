# Testing Guide

This guide provides comprehensive testing commands and procedures for the open.news application.

## Quick Test Commands

### 1. Database Seeding

Seed the database with a test user and popular sources:

```bash
# Seed with default test user (bsky.app)
go run cmd/seed.go

# Seed with a custom Bluesky handle
go run cmd/seed.go -handle your.handle.bsky.social

# Seed with custom handle and DID
go run cmd/seed.go -handle your.handle.bsky.social -did did:plc:your-actual-did
```

### 2. Start the Server

```bash
go run cmd/main.go server
```

Or using the Makefile:

```bash
make run
```

### 3. Test API Endpoints

#### Health Check
```bash
curl http://localhost:8080/health
```

#### Global Feed (AT Protocol endpoint)
```bash
curl "http://localhost:8080/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-global"
```

#### Personalized Feed (triggers follow import)
```bash
curl -H "Authorization: Bearer test-token" "http://localhost:8080/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-personal"
```

#### Internal API Endpoints
```bash
# Global feed (internal API)
curl http://localhost:8080/api/feeds/global

# Worker status
curl http://localhost:8080/api/worker/status
```

## Complete Testing Workflow

### 1. Environment Setup

Make sure you have PostgreSQL running and your `.env` file configured:

```bash
# Check if PostgreSQL is running
brew services list | grep postgresql

# If not running, start it
brew services start postgresql@15

# Verify database exists
psql -l | grep open_news
```

### 2. Database Preparation

```bash
# Run migrations
go run cmd/main.go migrate

# Seed test data
go run cmd/seed.go -handle your.test.handle
```

### 3. Server Testing

```bash
# Terminal 1: Start the server
go run cmd/main.go server

# Terminal 2: Run tests
make test-api
```

### 4. Feed Testing

#### Global Feed Testing
The global feed aggregates popular articles from all sources:

```bash
# Test empty state
curl -s http://localhost:8080/api/feeds/global | jq

# Test with pagination
curl -s "http://localhost:8080/api/feeds/global?limit=5&page=1" | jq

# Test AT Protocol endpoint
curl -s "http://localhost:8080/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-global" | jq
```

#### Personalized Feed Testing
The personalized feed requires authentication and triggers follow import:

```bash
# Test with mock auth (development mode)
curl -H "Authorization: Bearer test-token" -s "http://localhost:8080/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-personal" | jq

# This should trigger automatic import of follows for the test user
```

### 5. Worker Testing

```bash
# Check worker status
curl -s http://localhost:8080/api/worker/status | jq

# Expected response:
# {
#   "firehose_connected": true,
#   "workers_active": 3,
#   "articles_processed": 0,
#   "queue_size": 0
# }
```

## Testing Different User Scenarios

### Test User Creation

```bash
# Test with popular Bluesky accounts
go run cmd/seed.go -handle bsky.app -did did:plc:z72i7hdynmk6r22z27h6tvur
go run cmd/seed.go -handle jay.bsky.team -did did:plc:vpkhqolt662uhesyj6nxm7ys
go run cmd/seed.go -handle atproto.com -did did:plc:ewvi7nxzyoun6zhxrhs64oiz

# Test with your own handle (replace with your actual DID)
go run cmd/seed.go -handle your.handle.bsky.social -did your:actual:did
```

### Follow Import Testing

When a user visits their personalized feed, the system automatically attempts to import their follows from Bluesky. This happens in the background and you can monitor it in the logs.

## Development Testing

### Database Reset

```bash
# Drop and recreate database
dropdb open_news && createdb open_news

# Run migrations
go run cmd/main.go migrate

# Reseed
go run cmd/seed.go
```

### Clean Test Run

```bash
# Complete clean setup
make clean
make db-setup
make run-migrations
make seed
make run
```

## Production Testing

For production testing procedures, see [PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md).

## Troubleshooting

### Common Issues

1. **Database Connection Error**
   ```
   Error: failed to connect to database
   ```
   - Check if PostgreSQL is running: `brew services list | grep postgresql`
   - Verify database exists: `psql -l | grep open_news`
   - Check `.env` file configuration

2. **Seeder Fails to Import Follows**
   ```
   ⚠️ Could not fetch follows automatically: 401 Unauthorized
   ```
   - This is normal - follows will be imported when the user visits their personalized feed
   - No authentication is needed for seeding

3. **Feed Returns Empty Results**
   - Make sure you've seeded the database: `go run cmd/seed.go`
   - Check if the firehose worker is connected: `curl http://localhost:8080/api/worker/status`
   - Articles may take time to be discovered and processed

4. **Port Already in Use**
   ```
   Error: listen tcp :8080: bind: address already in use
   ```
   - Kill existing process: `lsof -ti:8080 | xargs kill -9`
   - Or use a different port: `PORT=8081 go run cmd/main.go server`

## Advanced Testing

### Load Testing

```bash
# Install hey (HTTP load testing tool)
brew install hey

# Test global feed endpoint
hey -n 100 -c 10 http://localhost:8080/api/feeds/global

# Test health endpoint
hey -n 1000 -c 50 http://localhost:8080/health
```

### Database Performance Testing

```bash
# Connect to database
psql open_news

# Check table sizes
\dt+

# Analyze query performance
EXPLAIN ANALYZE SELECT * FROM articles ORDER BY created_at DESC LIMIT 20;
```

## Continuous Integration

For automated testing in CI/CD pipelines, see the test commands in the `Makefile`:

```bash
make test          # Run all tests
make test-basic    # Basic functionality tests (no DB)
make test-api      # API endpoint tests
make test-db       # Database tests
```
