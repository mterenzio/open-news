#!/bin/bash

# Open News - Production Deployment Script
# This script helps you prepare and deploy your Bluesky Custom Feeds to production

set -e

echo "🚀 Open News - Production Deployment Assistant"
echo "=============================================="
echo ""

# Check if we're in production mode
PRODUCTION_MODE=${1:-""}
if [ "$PRODUCTION_MODE" != "production" ]; then
    echo "ℹ️  This script will help you prepare for production deployment."
    echo "   For actual deployment, run: ./deploy.sh production"
    echo ""
fi

# Environment check
echo "🔍 Environment Check"
echo "===================="

# Check Go version
if command -v go >/dev/null 2>&1; then
    GO_VERSION=$(go version | cut -d' ' -f3)
    echo "✅ Go: $GO_VERSION"
else
    echo "❌ Go not found - please install Go 1.22+"
    exit 1
fi

# Check for required files
echo ""
echo "📁 File Check"
echo "============="

REQUIRED_FILES=(
    "cmd/main.go"
    "internal/handlers/bluesky_feed.go"
    "internal/auth/jwt.go"
    "go.mod"
    "go.sum"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "❌ $file - missing required file"
        exit 1
    fi
done

echo ""
echo "🏗️  Build Test"
echo "=============="

# Test build
echo "Testing build..."
if go build -o bin/open-news-test cmd/main.go; then
    echo "✅ Build successful"
    rm -f bin/open-news-test
else
    echo "❌ Build failed"
    exit 1
fi

echo ""
echo "📋 Production Readiness Checklist"
echo "================================="

echo ""
echo "🌐 Domain & Infrastructure:"
echo "  [ ] Domain name registered"
echo "  [ ] Cloud hosting set up (AWS, GCP, DigitalOcean, etc.)"
echo "  [ ] SSL certificate configured"
echo "  [ ] PostgreSQL database in production"
echo ""

echo "🔐 Bluesky Account:"
echo "  [ ] Bluesky account created"
echo "  [ ] App password generated (Settings > Privacy & Security > App Passwords)"
echo "  [ ] Handle verified (optional but recommended)"
echo ""

echo "⚙️  Environment Configuration:"
echo "  [ ] .env.production file created"
echo "  [ ] Database URL configured"
echo "  [ ] Bluesky credentials set"
echo "  [ ] Domain name set"
echo ""

if [ "$PRODUCTION_MODE" = "production" ]; then
    echo "🚀 PRODUCTION DEPLOYMENT MODE"
    echo "=============================="
    
    # Check for production environment file
    if [ ! -f ".env.production" ]; then
        echo "❌ .env.production file not found"
        echo "   Please copy .env.production.template to .env.production and configure it"
        exit 1
    fi
    
    # Load production environment
    set -a
    source .env.production
    set +a
    
    echo "✅ Loaded production environment"
    
    # Validate required environment variables
    REQUIRED_ENV_VARS=(
        "DATABASE_URL"
        "DOMAIN"
        "BLUESKY_HANDLE"
        "BLUESKY_PASSWORD"
    )
    
    echo ""
    echo "🔍 Environment Variables Check:"
    for var in "${REQUIRED_ENV_VARS[@]}"; do
        if [ -n "${!var}" ]; then
            if [ "$var" = "DATABASE_URL" ] || [ "$var" = "BLUESKY_PASSWORD" ]; then
                echo "✅ $var: [REDACTED]"
            else
                echo "✅ $var: ${!var}"
            fi
        else
            echo "❌ $var: not set"
            exit 1
        fi
    done
    
    # Set production mode
    export GIN_MODE=release
    
    echo ""
    echo "🏗️  Building Production Binary"
    echo "============================="
    
    if go build -ldflags="-s -w" -o bin/open-news-prod cmd/main.go; then
        echo "✅ Production build successful"
    else
        echo "❌ Production build failed"
        exit 1
    fi
    
    echo ""
    echo "📄 Setting up DID Document"
    echo "=========================="
    
    # Create DID document
    mkdir -p static/.well-known
    cat > static/.well-known/did.json << EOF
{
  "@context": ["https://www.w3.org/ns/did/v1"],
  "id": "did:web:$DOMAIN",
  "service": [
    {
      "id": "#bsky_fg",
      "type": "BskyFeedGenerator",
      "serviceEndpoint": "https://$DOMAIN"
    }
  ]
}
EOF
    
    echo "✅ DID document created for $DOMAIN"
    
    echo ""
    echo "🐳 Docker Build (Optional)"
    echo "========================="
    
    if command -v docker >/dev/null 2>&1; then
        echo "Docker found. Build production image? (y/n)"
        read -r BUILD_DOCKER
        if [ "$BUILD_DOCKER" = "y" ] || [ "$BUILD_DOCKER" = "Y" ]; then
            echo "Building Docker image..."
            if docker build -f Dockerfile.prod -t open-news:latest .; then
                echo "✅ Docker image built successfully"
                echo "   Tag and push: docker tag open-news:latest your-registry/open-news:latest"
            else
                echo "❌ Docker build failed"
            fi
        fi
    else
        echo "Docker not found - skipping Docker build"
    fi
    
    echo ""
    echo "🔧 Database Migration Test"
    echo "========================="
    
    echo "Test database connection and run migrations? (y/n)"
    read -r TEST_DB
    if [ "$TEST_DB" = "y" ] || [ "$TEST_DB" = "Y" ]; then
        echo "Testing database connection..."
        if timeout 30 ./bin/open-news-prod migrate; then
            echo "✅ Database migrations successful"
        else
            echo "❌ Database migration failed"
            echo "   Please check your DATABASE_URL and database connectivity"
        fi
    fi
    
    echo ""
    echo "🎯 Feed URI Configuration"
    echo "========================"
    
    echo "Your feed URIs will be:"
    echo "  🌍 Global Feed:"
    echo "    at://did:web:$DOMAIN/app.bsky.feed.generator/open-news-global"
    echo ""
    echo "  👤 Personal Feed:"
    echo "    at://did:web:$DOMAIN/app.bsky.feed.generator/open-news-personal"
    echo ""
    
    echo "✅ PRODUCTION DEPLOYMENT READY!"
    echo ""
    echo "📝 Next Steps:"
    echo "1. Deploy to your cloud platform:"
    echo "   - Upload binary: bin/open-news-prod"
    echo "   - Upload static files: static/"
    echo "   - Set environment variables from .env.production"
    echo ""
    echo "2. Verify deployment:"
    echo "   curl https://$DOMAIN/health"
    echo "   curl https://$DOMAIN/.well-known/did.json"
    echo ""
    echo "3. Test feed endpoints:"
    echo "   curl \"https://$DOMAIN/xrpc/app.bsky.feed.describeFeedGenerator?feed=at://did:web:$DOMAIN/app.bsky.feed.generator/open-news-global\""
    echo ""
    echo "4. Register feeds with Bluesky (see PRODUCTION_DEPLOYMENT.md)"
    echo ""
    
else
    echo ""
    echo "🛠️  Setup Commands"
    echo "=================="
    echo ""
    echo "1. Create production environment:"
    echo "   cp .env.production.template .env.production"
    echo "   # Edit .env.production with your values"
    echo ""
    echo "2. Set up DID document:"
    echo "   ./scripts/setup-did.sh"
    echo ""
    echo "3. Test production build:"
    echo "   ./deploy.sh production"
    echo ""
    echo "📚 Read the complete guide:"
    echo "   cat PRODUCTION_DEPLOYMENT.md"
    echo ""
fi

echo "🎉 Deployment script complete!"
