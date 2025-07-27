#!/bin/bash

# Bluesky Feed Registration Script
# This script helps you register your custom feeds with Bluesky

set -e

echo "📡 Bluesky Feed Registration Assistant"
echo "====================================="
echo ""

# Check if required environment variables are set
if [ -z "$BLUESKY_HANDLE" ] || [ -z "$BLUESKY_PASSWORD" ] || [ -z "$DOMAIN" ]; then
    echo "❌ Missing required environment variables"
    echo ""
    echo "Please set the following environment variables:"
    echo "  export BLUESKY_HANDLE=your-handle.bsky.social"
    echo "  export BLUESKY_PASSWORD=your-app-password"
    echo "  export DOMAIN=your-domain.com"
    echo ""
    echo "Or load from .env.production:"
    echo "  source .env.production"
    echo ""
    exit 1
fi

echo "✅ Configuration loaded:"
echo "   Handle: $BLUESKY_HANDLE"
echo "   Domain: $DOMAIN"
echo ""

# Feed information
FEED_GENERATOR_DID="did:web:$DOMAIN"
GLOBAL_FEED_URI="at://$FEED_GENERATOR_DID/app.bsky.feed.generator/open-news-global"
PERSONAL_FEED_URI="at://$FEED_GENERATOR_DID/app.bsky.feed.generator/open-news-personal"

echo "🔗 Feed URIs to register:"
echo "   Global: $GLOBAL_FEED_URI"
echo "   Personal: $PERSONAL_FEED_URI"
echo ""

# Verify feeds are accessible
echo "🧪 Testing feed accessibility..."

if curl -f -s "https://$DOMAIN/health" > /dev/null; then
    echo "✅ Application is running"
else
    echo "❌ Cannot reach https://$DOMAIN/health"
    echo "   Please ensure your application is deployed and accessible"
    exit 1
fi

if curl -f -s "https://$DOMAIN/.well-known/did.json" > /dev/null; then
    echo "✅ DID document accessible"
else
    echo "❌ Cannot reach https://$DOMAIN/.well-known/did.json"
    echo "   Please ensure the DID document is properly deployed"
    exit 1
fi

# Test feed endpoints
echo ""
echo "Testing feed endpoints..."

if curl -f -s "https://$DOMAIN/xrpc/app.bsky.feed.describeFeedGenerator?feed=$GLOBAL_FEED_URI" > /dev/null; then
    echo "✅ Global feed endpoint working"
else
    echo "❌ Global feed endpoint not responding"
    exit 1
fi

if curl -f -s "https://$DOMAIN/xrpc/app.bsky.feed.describeFeedGenerator?feed=$PERSONAL_FEED_URI" > /dev/null; then
    echo "✅ Personal feed endpoint working"
else
    echo "❌ Personal feed endpoint not responding"
    exit 1
fi

echo ""
echo "🎯 Feed Registration Methods"
echo "==========================="
echo ""

echo "Method 1: Using Bluesky Web Interface"
echo "-------------------------------------"
echo "1. Go to https://bsky.app"
echo "2. Navigate to Settings > Moderation > Content Filtering"
echo "3. Click 'Add Custom Feed'"
echo "4. Enter your feed URI:"
echo "   Global: $GLOBAL_FEED_URI"
echo "   Personal: $PERSONAL_FEED_URI"
echo ""

echo "Method 2: Using AT Protocol CLI (Advanced)"
echo "------------------------------------------"
echo "1. Install the AT Protocol CLI tool"
echo "2. Create feed generator records:"
echo ""
echo "   # Global Feed"
echo "   at-cli put --pds https://bsky.social \\"
echo "     --repo \$BLUESKY_HANDLE \\"
echo "     --collection app.bsky.feed.generator \\"
echo "     --rkey open-news-global \\"
echo "     --record '{"
echo '       "$type": "app.bsky.feed.generator",'
echo '       "did": "'$FEED_GENERATOR_DID'",'
echo '       "displayName": "Open News - Global",'
echo '       "description": "Top stories from across the Bluesky network, ranked by engagement and quality.",'
echo '       "createdAt": "'$(date -u +%Y-%m-%dT%H:%M:%S.000Z)'"'
echo "     }'"
echo ""

echo "Method 3: Manual Testing"
echo "------------------------"
echo "Test your feeds by visiting these URLs:"
echo ""
echo "Global Feed Description:"
echo "https://$DOMAIN/xrpc/app.bsky.feed.describeFeedGenerator?feed=$GLOBAL_FEED_URI"
echo ""
echo "Global Feed Content:"
echo "https://$DOMAIN/xrpc/app.bsky.feed.getFeedSkeleton?feed=$GLOBAL_FEED_URI"
echo ""
echo "Personal Feed Description:"
echo "https://$DOMAIN/xrpc/app.bsky.feed.describeFeedGenerator?feed=$PERSONAL_FEED_URI"
echo ""

echo ""
echo "📝 Important Notes"
echo "=================="
echo ""
echo "🔐 Authentication:"
echo "   - Global feed works without authentication"
echo "   - Personal feed requires JWT token from Bluesky"
echo ""
echo "📊 Feed Discovery:"
echo "   - It may take time for feeds to appear in Bluesky's discovery"
echo "   - Users can manually add feeds using the URI"
echo "   - Consider sharing on social media to bootstrap users"
echo ""
echo "🔍 Debugging:"
echo "   - Check application logs for errors"
echo "   - Verify JWT token validation is working"
echo "   - Monitor feed request patterns"
echo ""

echo "📚 Additional Resources"
echo "======================"
echo ""
echo "- AT Protocol Feed Generator Docs: https://atproto.com/guides/applications#custom-feeds"
echo "- Bluesky Developer Discord: https://discord.gg/bluesky"
echo "- Feed Generator Examples: https://github.com/bluesky-social/feed-generator"
echo ""

echo "✅ Feed registration information complete!"
echo ""
echo "🎉 Your feeds are ready to be discovered on Bluesky!"
