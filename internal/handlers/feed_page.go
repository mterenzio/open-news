package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"open-news/internal/feeds"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FeedPageHandler handles web feed pages
type FeedPageHandler struct {
	feedService *feeds.FeedService
}

// NewFeedPageHandler creates a new feed page handler
func NewFeedPageHandler(db *gorm.DB) *FeedPageHandler {
	return &FeedPageHandler{
		feedService: feeds.NewFeedService(db),
	}
}

// ServeMainFeedPage serves the main feed page
func (h *FeedPageHandler) ServeMainFeedPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.File("static/feed.html")
}

// ServeGlobalFeedHTML serves the global feed as HTML
func (h *FeedPageHandler) ServeGlobalFeedHTML(c *gin.Context) {
	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	
	offset := (page - 1) * limit

	// Get the global feed
	feedResponse, err := h.feedService.GetGlobalFeed(limit, offset)
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusInternalServerError, `
			<div class="error-state">
				<i class="fas fa-exclamation-triangle"></i>
				<h3>Failed to load feed</h3>
				<p>%s</p>
			</div>
		`, err.Error())
		return
	}

	// Render HTML template
	h.renderFeedHTML(c, feedResponse, "Global Feed", "🌍", page, limit, "/feed/global")
}

// ServePersonalFeedHTML serves a personalized feed as HTML
func (h *FeedPageHandler) ServePersonalFeedHTML(c *gin.Context) {
	userIdentifier := c.Query("user")
	if userIdentifier == "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusBadRequest, `
			<div class="error-state">
				<i class="fas fa-user-slash"></i>
				<h3>User Required</h3>
				<p>Please provide a user handle or DID in the 'user' parameter.</p>
			</div>
		`)
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	
	offset := (page - 1) * limit

	// TODO: Implement personal feed service
	// For now, return global feed with user context
	feedResponse, err := h.feedService.GetGlobalFeed(limit, offset)
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusInternalServerError, `
			<div class="error-state">
				<i class="fas fa-exclamation-triangle"></i>
				<h3>Failed to load feed</h3>
				<p>%s</p>
			</div>
		`, err.Error())
		return
	}

	// Clean user identifier for display
	displayUser := userIdentifier
	if strings.HasPrefix(userIdentifier, "did:plc:") {
		displayUser = userIdentifier[:12] + "..."
	}

	// Render HTML template
	h.renderFeedHTML(c, feedResponse, "Personal Feed - "+displayUser, "👤", page, limit, "/feed/personal?user="+userIdentifier)
}

// ServeGlobalWidget serves the embeddable global feed widget
func (h *FeedPageHandler) ServeGlobalWidget(c *gin.Context) {
	h.serveWidget(c, "global", "")
}

// ServePersonalWidget serves the embeddable personal feed widget
func (h *FeedPageHandler) ServePersonalWidget(c *gin.Context) {
	userIdentifier := c.Query("user")
	h.serveWidget(c, "personal", userIdentifier)
}

// serveWidget serves embeddable widgets
func (h *FeedPageHandler) serveWidget(c *gin.Context, feedType string, userIdentifier string) {
	// Parse widget parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	theme := c.DefaultQuery("theme", "light")
	compact := c.DefaultQuery("compact", "false")
	autoRefresh, _ := strconv.Atoi(c.DefaultQuery("autorefresh", "300"))
	
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if autoRefresh < 60 {
		autoRefresh = 300 // Minimum 1 minute
	}

	// Get feed data
	feedResponse, err := h.feedService.GetGlobalFeed(limit, 0)
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusInternalServerError, `
			<div class="error-state">
				<i class="fas fa-exclamation-triangle"></i>
				<h3>Widget Error</h3>
				<p>Failed to load feed data</p>
			</div>
		`)
		return
	}

	// Determine widget title
	title := "Global News Feed"
	icon := "🌍"
	if feedType == "personal" && userIdentifier != "" {
		displayUser := userIdentifier
		if strings.HasPrefix(userIdentifier, "did:plc:") {
			displayUser = userIdentifier[:12] + "..."
		}
		title = "Personal Feed - " + displayUser
		icon = "👤"
	}

	// Widget CSS classes
	widgetClasses := "widget"
	if compact == "true" {
		widgetClasses += " compact"
	}

	// Render widget HTML
	widgetHTML := `
<!DOCTYPE html>
<html lang="en" data-theme="` + theme + `">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + `</title>
    <script src="https://unpkg.com/htmx.org@2.0.2"></script>
    <link rel="stylesheet" href="` + c.Request.Header.Get("Origin") + `/static/feed.css">
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css">
    <style>
        body { margin: 0; padding: 1rem; }
        .feed-header { margin-bottom: 1rem; }
        .feed-title { font-size: 1.25rem; }
    </style>
</head>
<body>
    <div class="` + widgetClasses + `">`

	// Add feed content
	widgetHTML += h.generateFeedHTML(feedResponse, title, icon, 1, limit, true, "")

	// Add auto-refresh script if enabled
	if autoRefresh > 0 {
		widgetHTML += `
    <script>
        setInterval(function() {
            location.reload();
        }, ` + strconv.Itoa(autoRefresh*1000) + `);
    </script>`
	}

	widgetHTML += `
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, widgetHTML)
}

// renderFeedHTML renders the feed HTML for the main page
func (h *FeedPageHandler) renderFeedHTML(c *gin.Context, feedResponse *feeds.FeedResponse, title, icon string, page, limit int, currentPath string) {
	html := h.generateFeedHTML(feedResponse, title, icon, page, limit, false, currentPath)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// generateFeedHTML generates HTML for feed content
func (h *FeedPageHandler) generateFeedHTML(feedResponse *feeds.FeedResponse, title, icon string, page, limit int, isWidget bool, currentPath string) string {
	html := `<div class="feed-header">
        <h1 class="feed-title">
            <span>` + icon + `</span>
            ` + title + `
        </h1>
        <div class="feed-meta">
            <div class="feed-stats">
                <i class="fas fa-newspaper"></i>
                <span>` + strconv.Itoa(len(feedResponse.Items)) + ` articles</span>
            </div>
            <div class="feed-stats">
                <i class="fas fa-clock"></i>
                <span>Updated ` + feedResponse.Meta.LastUpdatedAt.Format("Jan 2, 3:04 PM") + `</span>
            </div>`
	
	if !isWidget {
		html += `
            <button class="refresh-btn" 
                    hx-get="` + currentPath + `" 
                    hx-target="#feed-container"
                    hx-indicator="#loading">
                <i class="fas fa-sync-alt"></i> Refresh
            </button>`
	}
	
	html += `
        </div>
    </div>`

	if len(feedResponse.Items) == 0 {
		html += `
    <div class="empty-state">
        <i class="fas fa-newspaper"></i>
        <h3>No articles found</h3>
        <p>Check back later for new content or try seeding the database with sources.</p>
    </div>`
		return html
	}

	html += `<div class="feed-items">`

	for _, item := range feedResponse.Items {
		qualityClass := "low"
		if item.Article.QualityScore >= 0.7 {
			qualityClass = "high"
		} else if item.Article.QualityScore >= 0.5 {
			qualityClass = "medium"
		}

		// Format published time
		publishedTime := "Unknown"
		if item.Article.PublishedAt != nil {
			publishedTime = formatRelativeTime(*item.Article.PublishedAt)
		}

		// Safe title and description
		title := template.HTMLEscapeString(item.Article.Title)
		description := template.HTMLEscapeString(item.Article.Description)
		if len(description) > 200 {
			description = description[:200] + "..."
		}

		html += `
        <article class="feed-item">
            <div class="article-header">
                <div class="article-content">
                    <h2 class="article-title">
                        <a href="` + template.HTMLEscapeString(item.Article.URL) + `" target="_blank" rel="noopener">
                            ` + title + `
                        </a>
                    </h2>
                    <p class="article-description">` + description + `</p>
                </div>`
		
		if item.Article.ImageURL != "" {
			html += `
                <img src="` + template.HTMLEscapeString(item.Article.ImageURL) + `" 
                     alt="Article image" 
                     class="article-image"
                     loading="lazy">`
		}
		
		html += `
            </div>
            <div class="article-footer">
                <div class="source-info">`
		
		if item.Source.Avatar != "" {
			html += `
                    <img src="` + template.HTMLEscapeString(item.Source.Avatar) + `" 
                         alt="` + template.HTMLEscapeString(item.Source.DisplayName) + `" 
                         class="source-avatar">`
		} else {
			html += `<div class="source-avatar" style="background: var(--primary-color); display: flex; align-items: center; justify-content: center; color: white; font-weight: bold;">` + 
				string([]rune(item.Source.DisplayName)[0]) + `</div>`
		}
		
		html += `
                    <div class="source-details">
                        <div class="source-name">` + template.HTMLEscapeString(item.Source.DisplayName) + `</div>
                        <div class="source-handle">@` + template.HTMLEscapeString(item.Source.Handle) + `</div>
                    </div>
                </div>
                <div class="article-meta">
                    <span>` + publishedTime + `</span>
                    <div class="quality-score ` + qualityClass + `">
                        <i class="fas fa-star"></i>
                        ` + strconv.FormatFloat(item.Article.QualityScore, 'f', 1, 64) + `
                    </div>
                </div>
            </div>
        </article>`
	}

	html += `</div>`

	// Add pagination if not a widget
	if !isWidget && len(feedResponse.Items) == limit {
		html += h.generatePaginationHTML(page, limit, currentPath)
	}

	return html
}

// generatePaginationHTML generates pagination controls
func (h *FeedPageHandler) generatePaginationHTML(currentPage, limit int, currentPath string) string {
	html := `<div class="pagination">`
	
	if currentPage > 1 {
		html += `
        <button hx-get="` + currentPath + `?page=` + strconv.Itoa(currentPage-1) + `&limit=` + strconv.Itoa(limit) + `" 
                hx-target="#feed-container"
                hx-indicator="#loading">
            <i class="fas fa-chevron-left"></i> Previous
        </button>`
	}
	
	html += `<span class="current-page">Page ` + strconv.Itoa(currentPage) + `</span>`
	
	html += `
        <button hx-get="` + currentPath + `?page=` + strconv.Itoa(currentPage+1) + `&limit=` + strconv.Itoa(limit) + `" 
                hx-target="#feed-container"
                hx-indicator="#loading">
            Next <i class="fas fa-chevron-right"></i>
        </button>`
	
	html += `</div>`
	return html
}

// Helper functions
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	
	if diff < time.Minute {
		return "Just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return strconv.Itoa(minutes) + " minutes ago"
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return strconv.Itoa(hours) + " hours ago"
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return strconv.Itoa(days) + " days ago"
	} else {
		return t.Format("Jan 2")
	}
}
