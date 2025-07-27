# open.news - Social News Aggregation Platform

# open.news - Social News Aggregation Platform

open.news is an advanced social news aggregation platform built on top of Bluesky. It intelligently aggregates, ranks, and curates articles shared across the Bluesky network using sophisticated quality scoring algorithms and engagement metrics.

## Features

- **User Management**: Bluesky users can sign up simply by visiting a custom feed
- **Source Tracking**: Tracks users who share links and their engagement metrics
- **Real-time Monitoring**: Consumes Bluesky firehose to monitor articles shared by followed sources
- **Article Caching**: Canonical URL storage with JSON-LD and Open Graph metadata
- **AI-Powered Facts**: Extracts facts from articles with OpenAI embeddings
- **Background Workers**: Automated processing of articles and feed updates
- **Dual Feeds**: 
  - Global top stories feed
  - Personalized feed based on user's followed sources

## Architecture

### Data Models

- **Users**: Bluesky users who sign up by visiting a custom feed
- **Sources**: Users that share links (content creators)
- **User Sources**: Relationship between users and sources they follow
- **Articles**: Canonical URLs with metadata and cached HTML content
- **Source Articles**: Posts/reposts from sources containing articles
- **Article Facts**: AI-extracted facts with OpenAI embeddings
- **Feeds**: Custom feeds (global and personalized)

### Tech Stack

- **Backend**: Go (Golang) with Gin web framework
- **Database**: PostgreSQL with GORM
- **Real-time Processing**: WebSocket connection to Bluesky firehose
- **Background Jobs**: Goroutine-based workers for article processing
- **External APIs**: 
  - Bluesky AT Protocol
  - OpenAI API (for embeddings)
- **Caching**: Article HTML caching to minimize requests

## Getting Started

### Quick Start for macOS (Recommended)

1. **Test basic functionality** (no database required):
   ```bash
   make test-basic
   ```

2. **Install and setup PostgreSQL**:
   ```bash
   make db-install    # Install PostgreSQL with Homebrew
   make db-setup      # Create database
   make setup-env     # Create .env file
   ```

3. **Run the application**:
   ```bash
   make run
   ```

4. **Seed with example sources** (in another terminal):
   ```bash
   make seed
   ```

5. **Test the API**:
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/api/worker/status
   curl http://localhost:8080/api/feeds/global
   ```

6. **Access the developer dashboard**: Visit http://localhost:8080 for testing commands and documentation links

For detailed testing procedures, see [TESTING.md](TESTING.md).

## Development Tools

### Quick Server Management

Use the development script for convenient server management:

```bash
# Restart server (kills old process and starts new one)
./dev restart

# Check server status and health
./dev status

# Follow server logs
./dev logs

# Seed database with custom user
./dev seed -handle your.handle.bsky.social

# Test API endpoints
./dev test

# Show all available commands
./dev help
```

### Makefile Commands

```bash
# Server management
make restart        # Restart server
make stop          # Stop server  
make status        # Check status

# Testing
make test-api      # Test API endpoints
make test-feeds    # Test feed endpoints
make test-db       # Test database

# Development
make run           # Start server
make seed          # Seed database
make migrate       # Run migrations
```

### Alternative: Docker Setup

If you have Docker installed, see [DEVELOPMENT.md](DEVELOPMENT.md) for Docker instructions.

### Manual Setup

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed setup instructions.

## Web Interface

When running the server, visit http://localhost:8080 to access the developer dashboard with:

- 📚 **Documentation Links**: Quick access to all project documentation
- 🧪 **Testing Commands**: Copy-paste commands for testing the API
- 🔗 **Live API Links**: Direct links to test endpoints
- ⚡ **Quick Start Guide**: Essential commands for development

The web interface provides an easy way to test and explore the API without remembering complex curl commands.

## API Endpoints

### Feeds

- `GET /api/feeds/global` - Get global top stories feed
- `GET /api/feeds/personalized` - Get personalized feed (requires authentication)

### Workers

- `GET /api/worker/status` - Get background worker status

### Health Check

- `GET /health` - Health check endpoint

### Query Parameters

Both feed endpoints support:
- `limit`: Number of items to return (max 100, default 20)
- `page`: Page number for pagination (default 1)

## Database Schema

The application uses PostgreSQL with the following main tables:

- `users` - Bluesky users
- `sources` - Content creators who share links
- `user_sources` - Many-to-many relationship between users and sources
- `articles` - Cached articles with metadata
- `source_articles` - Posts containing articles
- `article_facts` - AI-extracted facts with embeddings
- `feeds` - Feed configurations
- `feed_items` - Articles in feeds with rankings

## Development

### Project Structure

```
open-news/
├── cmd/                    # Application entry points
│   └── main.go
├── internal/               # Internal application code
│   ├── models/            # Data models
│   ├── handlers/          # HTTP handlers
│   ├── database/          # Database connection and migrations
│   ├── bluesky/           # Bluesky API client and firehose consumer
│   ├── feeds/             # Feed service logic
│   └── worker/            # Background workers and article fetcher
├── migrations/            # Database migrations
├── .env.example          # Environment configuration template
├── go.mod                # Go module definition
└── README.md             # This file
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

[License information to be added]

## Roadmap

### MVP Features ✅ COMPLETED
- [x] Basic data models
- [x] Database schema with PostgreSQL
- [x] RESTful API endpoints
- [x] Bluesky firehose integration
- [x] Real-time article discovery from followed sources
- [x] Background worker system with graceful shutdown
- [x] Article content fetching and caching
- [x] Docker development environment
- [x] Database seeding utilities

### Next Phase (In Development)
- [ ] Feed ranking algorithms
- [ ] OpenAI integration for fact extraction
- [ ] User authentication (Bluesky-based)
- [ ] Enhanced metadata extraction (JSON-LD, Open Graph)
- [ ] Engagement metrics tracking from Bluesky

### Future Features
- [ ] Real-time feed updates
- [ ] Advanced ranking algorithms
- [ ] Article summarization
- [ ] Topic categorization
- [ ] User preferences and filtering
- [ ] Analytics and metrics
- [ ] Mobile app
- [ ] Browser extension
