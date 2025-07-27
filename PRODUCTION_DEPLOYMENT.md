# 🚀 Production Deployment Guide for Open News Bluesky Custom Feeds

## Prerequisites Checklist

### ✅ **Domain & Infrastructure**
- [ ] **Domain name** registered (e.g., `opennews.app`)
- [ ] **SSL certificate** configured
- [ ] **Cloud hosting** set up (AWS, GCP, DigitalOcean, etc.)
- [ ] **PostgreSQL database** in production environment
- [ ] **Environment variables** configured

### ✅ **Bluesky Account Setup**
- [ ] **Bluesky account** for your feed generator
- [ ] **App password** generated (not your main password!)
- [ ] **Handle verification** (optional but recommended)

---

## Step 1: Environment Configuration

### 1.1 Create Production Environment File

Create `.env.production`:

```env
# Database Configuration
DATABASE_URL=postgres://username:password@your-db-host:5432/opennews?sslmode=require

# Server Configuration
GIN_MODE=release
PORT=8080

# Bluesky Configuration
BLUESKY_HANDLE=your-handle.bsky.social
BLUESKY_PASSWORD=your-app-password-here
BLUESKY_BASE_URL=https://bsky.social

# Feed Generator Configuration
DOMAIN=opennews.app
FEED_GENERATOR_DID=did:web:opennews.app

# Security (generate strong random strings)
JWT_SECRET=your-super-secret-jwt-key-here
```

### 1.2 Update Main Application Config

**Action Required**: Update your main.go to use production JWT verifier when in production mode.

---

## Step 2: Production Code Changes

### 2.1 Switch to Production JWT Verification

In `internal/handlers/bluesky_feed.go`, we need to conditionally use real JWT verification:

```go
// In NewBlueSkyFeedHandler function
func NewBlueSkyFeedHandler(db *gorm.DB, blueskyClient *bluesky.Client) *BlueSkyFeedHandler {
    var jwtVerifier interface {
        ValidateToken(authHeader string) (string, bool)
        ExtractDIDFromToken(tokenString string) (string, error)
    }
    
    // Use real JWT verification in production
    if os.Getenv("GIN_MODE") == "release" {
        jwtVerifier = auth.NewJWTVerifier()
    } else {
        jwtVerifier = auth.NewMockJWTVerifier()
    }
    
    return &BlueSkyFeedHandler{
        db:            db,
        feedService:   feeds.NewFeedService(db),
        blueskyClient: blueskyClient,
        jwtVerifier:   jwtVerifier,
    }
}
```

### 2.2 Update Feed URIs for Production

Replace placeholder DIDs with your actual domain-based DID.

---

## Step 3: Build & Test Locally

### 3.1 Build Production Binary

```bash
# Build for your target platform
go build -o bin/open-news-prod cmd/main.go

# Test with production environment
cp .env.production .env
./bin/open-news-prod server
```

### 3.2 Test Endpoints

```bash
# Test feed descriptions
curl "https://your-domain.com/xrpc/app.bsky.feed.describeFeedGenerator?feed=at://did:web:your-domain.com/app.bsky.feed.generator/open-news-global"

# Test feed skeleton
curl "https://your-domain.com/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:web:your-domain.com/app.bsky.feed.generator/open-news-global"
```

---

## Step 4: Database Setup

### 4.1 Production Database Migration

```bash
# Run migrations on production database
DATABASE_URL="your-production-db-url" ./bin/open-news-prod migrate

# Seed initial data (optional)
DATABASE_URL="your-production-db-url" make seed
```

### 4.2 Database Security

- [ ] **Enable SSL** for database connections
- [ ] **Set up backups** and monitoring
- [ ] **Configure connection pooling**
- [ ] **Set up read replicas** (if needed)

---

## Step 5: Deploy to Cloud Platform

### 5.1 Docker Deployment (Recommended)

Create `Dockerfile.prod`:

```dockerfile
FROM golang:1.22.1-alpine AS builder

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . .
RUN go build -o bin/open-news cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/bin/open-news .
COPY --from=builder /app/static ./static

EXPOSE 8080
CMD ["./open-news", "server"]
```

### 5.2 Cloud Platform Specific Steps

#### **For Railway/Render/Heroku:**
```bash
# Deploy directly from Git
git push production main
```

#### **For AWS/GCP/DigitalOcean:**
```bash
# Build and push Docker image
docker build -f Dockerfile.prod -t open-news:latest .
docker tag open-news:latest your-registry/open-news:latest
docker push your-registry/open-news:latest
```

---

## Step 6: Configure DID Document

### 6.1 Create DID Document

Your domain must serve a DID document at `https://your-domain.com/.well-known/did.json`:

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

### 6.2 Serve Static Files

Ensure your server serves the `.well-known` directory:

```go
// Add to main.go
r.Static("/.well-known", "./static/.well-known")
```

---

## Step 7: Register with Bluesky

### 7.1 Create Feed Generator Record

You need to publish a feed generator record to the AT Protocol network. This requires:

1. **AT Protocol client** (you can use the Bluesky CLI or API)
2. **Your Bluesky credentials**
3. **Feed metadata**

### 7.2 Feed Registration Script

Create `scripts/register-feeds.go`:

```go
// Script to register feeds with Bluesky
// This will create the official feed generator records
```

---

## Step 8: Monitoring & Security

### 8.1 Set Up Monitoring

- [ ] **Application logs** (structured logging)
- [ ] **Error tracking** (Sentry, Rollbar)
- [ ] **Performance monitoring** (New Relic, DataDog)
- [ ] **Health checks** and uptime monitoring

### 8.2 Security Configuration

- [ ] **Rate limiting** for feed endpoints
- [ ] **CORS configuration** for allowed origins
- [ ] **Security headers** (CSP, HSTS, etc.)
- [ ] **Input validation** and sanitization

### 8.3 Backup Strategy

- [ ] **Database backups** (automated)
- [ ] **Application state** backup
- [ ] **Disaster recovery** plan

---

## Step 9: Load Testing

### 9.1 Test Feed Performance

```bash
# Test with Apache Bench or similar
ab -n 1000 -c 10 "https://your-domain.com/xrpc/app.bsky.feed.getFeedSkeleton?feed=at://did:web:your-domain.com/app.bsky.feed.generator/open-news-global"
```

### 9.2 Database Performance

- [ ] **Query optimization**
- [ ] **Index optimization**
- [ ] **Connection pool tuning**

---

## Step 10: Go Live

### 10.1 DNS Configuration

- [ ] **A/AAAA records** pointing to your server
- [ ] **CNAME for www** (if needed)
- [ ] **TTL configuration**

### 10.2 SSL Certificate

- [ ] **Let's Encrypt** or commercial SSL
- [ ] **Auto-renewal** configured
- [ ] **HTTPS redirect** enabled

### 10.3 Final Verification

```bash
# Test all endpoints
curl -I "https://your-domain.com/health"
curl "https://your-domain.com/.well-known/did.json"
curl "https://your-domain.com/xrpc/app.bsky.feed.describeFeedGenerator?feed=at://did:web:your-domain.com/app.bsky.feed.generator/open-news-global"
```

---

## 🎉 Post-Deployment

### Marketing & Distribution

1. **Announce on Bluesky** - Share your custom feeds
2. **Community engagement** - Get initial users
3. **Feed optimization** - Monitor usage and improve algorithms
4. **User feedback** - Iterate based on real usage

### Ongoing Maintenance

- [ ] **Regular updates** and security patches
- [ ] **Performance optimization**
- [ ] **Feature enhancements**
- [ ] **User support**

---

## 🚨 Troubleshooting

### Common Issues

1. **Feed not appearing in Bluesky**: Check DID document and registration
2. **Authentication errors**: Verify JWT implementation and Bluesky keys
3. **Performance issues**: Check database queries and caching
4. **Rate limiting**: Monitor Bluesky API usage

### Support Resources

- [AT Protocol Documentation](https://atproto.com/)
- [Bluesky Developer Discord](https://discord.gg/bluesky)
- [Feed Generator Examples](https://github.com/bluesky-social/feed-generator)

---

**Ready to deploy? Let's start with Step 1! 🚀**
