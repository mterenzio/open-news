# 🚀 Quick Production Deployment Guide

## Step-by-Step Deployment Process

### 1. **Prepare Environment** (5 minutes)

```bash
# Copy and configure production environment
cp .env.production.template .env.production

# Edit .env.production with your values:
# - DATABASE_URL: Your production PostgreSQL database
# - DOMAIN: Your domain name (e.g., opennews.app)
# - BLUESKY_HANDLE: Your Bluesky handle
# - BLUESKY_PASSWORD: Your Bluesky app password
```

### 2. **Set Up DID Document** (2 minutes)

```bash
# Create DID document for your domain
./scripts/setup-did.sh
```

### 3. **Test Production Build** (3 minutes)

```bash
# Test everything works
./deploy.sh production
```

### 4. **Deploy to Your Platform** (10-30 minutes)

Choose one deployment method:

#### **Option A: Docker (Recommended)**
```bash
# Build Docker image
docker build -f Dockerfile.prod -t open-news:latest .

# Tag and push to your registry
docker tag open-news:latest your-registry/open-news:latest
docker push your-registry/open-news:latest

# Deploy to your cloud platform
```

#### **Option B: Binary Deployment**
```bash
# Upload these files to your server:
# - bin/open-news-prod (the binary)
# - static/ (directory with DID document)
# - Set environment variables from .env.production
```

### 5. **Verify Deployment** (5 minutes)

```bash
# Test your deployed application
curl https://your-domain.com/health
curl https://your-domain.com/.well-known/did.json

# Test feed endpoints
curl "https://your-domain.com/xrpc/app.bsky.feed.describeFeedGenerator?feed=at://did:web:your-domain.com/app.bsky.feed.generator/open-news-global"
```

### 6. **Register Feeds with Bluesky** (10 minutes)

```bash
# Load your production environment
source .env.production

# Run registration assistant
./scripts/register-feeds.sh
```

---

## 🎯 Your Feed URIs

Once deployed, your custom feeds will be available at:

- **Global Feed**: `at://did:web:your-domain.com/app.bsky.feed.generator/open-news-global`
- **Personal Feed**: `at://did:web:your-domain.com/app.bsky.feed.generator/open-news-personal`

---

## 🚨 Troubleshooting

### Common Issues

**Build fails**: Check Go version (needs 1.22+)
```bash
go version
```

**Database connection fails**: Verify DATABASE_URL format
```bash
# Should be: postgres://user:pass@host:port/dbname?sslmode=require
```

**DID document not accessible**: Check static file serving
```bash
curl https://your-domain.com/.well-known/did.json
```

**Feeds not working**: Check authentication and feed URIs
```bash
# Check application logs for JWT and database errors
```

---

## 📋 Production Checklist

### Before Deployment
- [ ] Domain registered and SSL configured
- [ ] PostgreSQL database set up
- [ ] Bluesky account and app password ready
- [ ] .env.production configured
- [ ] DID document created
- [ ] Production build tested

### After Deployment
- [ ] Health endpoint responds: `/health`
- [ ] DID document accessible: `/.well-known/did.json`
- [ ] Feed endpoints working: `/xrpc/app.bsky.feed.*`
- [ ] Database migrations applied
- [ ] Background workers running
- [ ] Monitoring set up

### Feed Registration
- [ ] Feeds accessible via AT Protocol
- [ ] JWT authentication working for personal feeds
- [ ] Feed URIs registered with Bluesky
- [ ] Test users can subscribe to feeds

---

## 📞 Support

If you encounter issues:

1. **Check logs** for error messages
2. **Review** `PRODUCTION_DEPLOYMENT.md` for detailed guide
3. **Test locally** with `./deploy.sh` first
4. **Verify environment** variables are correct

---

**Total Time**: ~30-60 minutes for complete deployment

**Result**: Production-ready Bluesky Custom Feeds! 🎉
