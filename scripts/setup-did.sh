#!/bin/bash

# Setup DID Document for Production
# This script creates the necessary DID document for Bluesky custom feeds

set -e

echo "🔧 Setting up DID Document for Production"
echo "=========================================="

# Get domain from environment or prompt user
DOMAIN=${DOMAIN:-}
if [ -z "$DOMAIN" ]; then
    echo "Enter your domain name (e.g., opennews.app):"
    read -r DOMAIN
fi

if [ -z "$DOMAIN" ]; then
    echo "❌ Domain is required"
    exit 1
fi

echo "📍 Domain: $DOMAIN"

# Create static directory structure
mkdir -p static/.well-known

# Create DID document
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

echo "✅ Created DID document at static/.well-known/did.json"

# Validate JSON
if command -v jq > /dev/null 2>&1; then
    echo "🔍 Validating DID document JSON..."
    if jq empty static/.well-known/did.json; then
        echo "✅ DID document JSON is valid"
    else
        echo "❌ DID document JSON is invalid"
        exit 1
    fi
else
    echo "⚠️  jq not found, skipping JSON validation"
fi

# Display the DID document
echo ""
echo "📄 Your DID Document:"
echo "===================="
cat static/.well-known/did.json

echo ""
echo "🔗 Your Feed URIs will be:"
echo "  Global Feed: at://did:web:$DOMAIN/app.bsky.feed.generator/open-news-global"
echo "  Personal Feed: at://did:web:$DOMAIN/app.bsky.feed.generator/open-news-personal"

echo ""
echo "📝 Next Steps:"
echo "1. Deploy your application to https://$DOMAIN"
echo "2. Ensure https://$DOMAIN/.well-known/did.json is accessible"
echo "3. Update your environment variables with DOMAIN=$DOMAIN"
echo "4. Test the DID document: curl https://$DOMAIN/.well-known/did.json"
echo "5. Register your feeds with Bluesky"

echo ""
echo "✅ DID document setup complete!"
