<!-- Use this file to provide workspace-specific custom instructions to Copilot. For more details, visit https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file -->

# open.news - Copilot Instructions

This is a Go project for an advanced social news aggregation platform built on top of Bluesky. The application intelligently aggregates, ranks, and curates articles shared across the Bluesky network using sophisticated quality scoring algorithms.

## Project Context

- **Language**: Go (Golang) 1.22.1
- **Framework**: Gin web framework for HTTP API
- **Database**: PostgreSQL with GORM ORM
- **External APIs**: Bluesky AT Protocol, OpenAI API
- **Architecture**: Clean architecture with internal packages

## Key Components

1. **Data Models** (`internal/models/`): GORM models for users, sources, articles, feeds
2. **Database** (`internal/database/`): Database connection and migration management
3. **Handlers** (`internal/handlers/`): HTTP request handlers using Gin
4. **Feeds** (`internal/feeds/`): Feed generation and ranking logic
5. **Bluesky** (`internal/bluesky/`): Bluesky API client and integration

## Code Style Guidelines

- Follow Go conventions and best practices
- Use GORM tags for database schema definitions
- Include comprehensive error handling
- Use structured logging
- Follow RESTful API design principles
- Use dependency injection for services
- Include proper JSON tags for API responses

## Database Considerations

- Use UUID primary keys
- Include created_at and updated_at timestamps
- Use proper foreign key relationships
- Consider indexing for performance
- Use PostgreSQL-specific features where beneficial (arrays, JSONB)

## API Design

- Use consistent response formats
- Include pagination for list endpoints
- Provide meaningful HTTP status codes
- Include metadata in responses (pagination info, etc.)
- Use proper error response structures

## Bluesky Integration

- Handle AT Protocol URIs and CIDs correctly
- Respect rate limits
- Cache user profiles and posts appropriately
- Extract links from posts using facets and embeds

## Security Considerations

- Validate all user inputs
- Use environment variables for sensitive configuration
- Implement proper authentication middleware
- Sanitize database queries (GORM helps with this)

When suggesting code changes or new features, please consider these patterns and maintain consistency with the existing codebase.
