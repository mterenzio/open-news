# Bluesky Custom Feeds - Implementation Guide

## Overview

Your open.news application now supports **Bluesky Custom Feeds**! This allows users to discover your feeds through Bluesky's official interface and automatically creates user accounts based on their Bluesky follows.

## 🎯 What You've Built

### Two Custom Feeds

1. **open.news - Global**: Top stories from across the Bluesky network, ranked by engagement and quality
2. **open.news - Personal**: Personalized news feed based on accounts the user follows on Bluesky

### Automatic User Onboarding

When a user subscribes to your personalized feed:
1. **User Creation**: Automatically creates a user record from their Bluesky DID
2. **Follow Import**: Imports all their Bluesky follows as news sources
3. **Personalized Content**: Serves articles only from sources they follow
4. **Profile Sync**: Updates user profiles with latest Bluesky information

## 🔧 API Endpoints

### Feed Skeleton Endpoint
```
GET /xrpc/app.bsky.feed.getFeedSkeleton?feed={FEED_URI}
```

**Supported Feeds:**
- `at://did:plc:your-did/app.bsky.feed.generator/open-news-global`
- `at://did:plc:your-did/app.bsky.feed.generator/open-news-personal`

**Response Format:**
```json
{
  "feed": [
    {
      "post": {
        "uri": "at://did:plc:example/app.bsky.feed.post/abc123",
        "cid": "bafyrei...",
        "author": {
          "did": "did:plc:example",
          "handle": "user.bsky.social",
          "displayName": "User Name",
          "avatar": "https://..."
        },
        "record": {
          "$type": "app.bsky.feed.post",
          "text": "Article title...",
          "createdAt": "2025-07-27T08:53:21.000Z",
          "embed": {
            "$type": "app.bsky.embed.external",
            "external": {
              "uri": "https://example.com/article",
              "title": "Article Title",
              "description": "Article description",
              "thumb": "https://example.com/image.jpg"
            }
          }
        },
        "indexedAt": "2025-07-27T08:53:21.000Z"
      }
    }
  ],
  "cursor": "1722073401"
}
```

### Feed Description Endpoint
```
GET /xrpc/app.bsky.feed.describeFeedGenerator?feed={FEED_URI}
```

**Response Format:**
```json
{
  "uri": "at://did:plc:your-did/app.bsky.feed.generator/open-news-global",
  "displayName": "open.news - Global",
  "description": "Top stories from across the Bluesky network, ranked by engagement and quality.",
  "avatar": "https://your-domain.com/feed-avatar.jpg",
  "createdBy": "did:plc:your-feed-generator-did"
}
```

## 🚀 Publishing Your Feeds to Bluesky

### Step 1: Create a Feed Generator Record

You need to publish a feed generator record to the AT Protocol network. Here's the process:

1. **Set up Bluesky credentials** in your `.env`:
   ```env
   BLUESKY_IDENTIFIER=your-handle.bsky.social
   BLUESKY_PASSWORD=your-app-password
   ```

2. **Create the feed generator record** (you'll need to implement this):
   ```typescript
   // This is pseudocode - you'll need to implement in Go or use the Bluesky CLI
   const feedRecord = {
     "$type": "app.bsky.feed.generator",
     "did": "did:web:your-domain.com",
     "displayName": "open.news - Global",
     "description": "Top stories from across the Bluesky network",
     "avatar": "https://your-domain.com/feed-avatar.jpg",
     "createdAt": new Date().toISOString()
   };
   ```

### Step 2: Deploy to Production

1. **Domain Setup**: Deploy your application to a public domain
2. **SSL Certificate**: Ensure HTTPS is enabled
3. **Update Feed URIs**: Replace `did:plc:example` with your actual DID

### Step 3: Configure Your DID

Set up a `did:web` DID document at `https://your-domain.com/.well-known/did.json`:

```json
{
  "@context": ["https://www.w3.org/ns/did/v1"],
  "id": "did:web:your-domain.com",
  "service": [
    {
      "id": "#bsky_fg",
      "type": "BskyFeedGenerator",
      "serviceEndpoint": "https://your-domain.com"
    }
  ]
}
```

## 🔐 Authentication & JWT Verification

The current implementation has a placeholder for JWT verification. For production, you need to:

### 1. Install JWT Library
```bash
go get github.com/golang-jwt/jwt/v5
```

### 2. Implement JWT Verification

Replace the placeholder `extractDIDFromJWT` function:

```go
func (h *BlueSkyFeedHandler) extractDIDFromJWT(tokenString string) string {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        // Verify signing method
        if _, ok := token.Method.(*jwt.SigningMethodRS256); !ok {
            return nil, fmt.Errorf("unexpected signing method")
        }
        
        // Get Bluesky's public key for verification
        return getBlueskyPublicKey(), nil
    })
    
    if err != nil || !token.Valid {
        return ""
    }
    
    if claims, ok := token.Claims.(jwt.MapClaims); ok {
        if sub, ok := claims["sub"].(string); ok {
            return sub // This is the user's DID
        }
    }
    
    return ""
}
```

### 3. Test Authentication

```bash
# Test with Authorization header
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:8080/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-personal"
```

## 🔄 User Follow Import

When a user visits your personalized feed:

1. **User Detection**: Extract DID from JWT token
2. **User Creation**: Create user record if doesn't exist
3. **Follow Import**: Fetch all accounts they follow via `app.bsky.graph.getFollows`
4. **Source Creation**: Create source records for each follow
5. **Relationship Mapping**: Create user-source relationships
6. **Feed Generation**: Return articles from their followed sources

## 📊 Testing Your Implementation

### 1. Test Feed Discovery
```bash
# Global feed
curl "http://localhost:8080/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-global"

# Personal feed (requires auth)
curl -H "Authorization: Bearer JWT_TOKEN" \
  "http://localhost:8080/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:plc:example/app.bsky.feed.generator/open-news-personal"
```

### 2. Test Feed Metadata
```bash
curl "http://localhost:8080/xrpc/app.bsky.feed.describeFeedGenerator?feed=at://did:plc:example/app.bsky.feed.generator/open-news-global"
```

### 3. Verify User Creation
Check your database after a user visits the personalized feed:
```sql
SELECT * FROM users ORDER BY created_at DESC LIMIT 5;
SELECT * FROM sources ORDER BY created_at DESC LIMIT 10;
SELECT * FROM user_sources ORDER BY created_at DESC LIMIT 10;
```

## 🎨 Customization Options

### Feed Names and Descriptions
Update in `internal/handlers/bluesky_feed.go`:
```go
feedInfo = map[string]interface{}{
    "displayName": "Your Custom Feed Name",
    "description": "Your custom description",
    "avatar": "https://your-domain.com/avatar.jpg",
}
```

### Feed Logic
Modify `getFilteredGlobalFeed` and personalized feed logic to:
- Change ranking algorithms
- Add content filtering
- Implement different recommendation strategies

### User Import Behavior
Customize `importUserFollows` to:
- Filter which follows to import
- Set custom quality scores
- Add rate limiting

## 🚦 Production Considerations

### 1. Rate Limiting
Implement rate limiting for feed requests:
```go
// Add rate limiting middleware
func rateLimitMiddleware() gin.HandlerFunc {
    // Implementation depends on your rate limiting library
}
```

### 2. Caching
Cache feed responses to improve performance:
```go
// Redis or in-memory cache for feed responses
func (h *BlueSkyFeedHandler) getCachedFeed(feedURI string) (*ATProtoFeedResponse, bool) {
    // Check cache first
}
```

### 3. Monitoring
Monitor feed usage and performance:
- Request counts per feed
- Response times
- User growth from feeds
- Error rates

### 4. Content Moderation
Implement content filtering:
- Block inappropriate sources
- Filter spam articles
- Implement user reporting

## 🎉 Next Steps

1. **Add seed data**: `make seed` to populate example sources
2. **Deploy to production**: Set up domain and SSL
3. **Register feeds**: Submit to Bluesky's feed registry
4. **Monitor usage**: Track user adoption and engagement
5. **Iterate**: Improve ranking and recommendation algorithms

Your Bluesky Custom Feeds are now ready to bridge the gap between Bluesky's social graph and curated news experiences! 🚀

---

## 📚 Additional Resources

- [AT Protocol Feed Generator Specification](https://atproto.com/specs/feed-generator)
- [Bluesky Feed Generator Examples](https://github.com/bluesky-social/feed-generator)
- [AT Protocol Documentation](https://atproto.com/)
- [Bluesky API Reference](https://docs.bsky.app/)
