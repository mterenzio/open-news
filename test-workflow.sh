#!/bin/bash

# Test Development Workflow
# This demonstrates the typical development reset cycle

echo "🧪 Open News Development Workflow Test"
echo "======================================="
echo ""

echo "1. Current database state:"
psql postgresql://mterenzi:@localhost:5432/open_news -c "
    SELECT 
        (SELECT COUNT(*) FROM users) as users,
        (SELECT COUNT(*) FROM sources) as sources,
        (SELECT COUNT(*) FROM user_sources) as user_sources,
        (SELECT COUNT(*) FROM articles) as articles;
"

echo ""
echo "2. This workflow would:"
echo "   a) ./dev reset        # Clear all data"
echo "   b) go run cmd/seed.go -handle your.handle.bsky.social"
echo "      # Create user + import real follows"
echo ""

echo "✅ Development workflow ready!"
echo ""
echo "💡 Usage examples:"
echo "  ./dev reset && go run cmd/seed.go -handle librenews.bsky.social"
echo "  ./dev reset && go run cmd/seed.go -handle some.other.handle"
echo "  ./dev reset && go run cmd/seed.go -handle test.user  # Uses mock data"
