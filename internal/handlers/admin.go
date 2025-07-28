package handlers

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"open-news/internal/models"
	"open-news/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminHandler handles admin interface
type AdminHandler struct {
	db                 *gorm.DB
	userFollowsService *services.UserFollowsService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(db *gorm.DB, userFollowsService *services.UserFollowsService) *AdminHandler {
	return &AdminHandler{
		db:                 db,
		userFollowsService: userFollowsService,
	}
}

// AdminAuth middleware for basic password protection
func (h *AdminHandler) AdminAuth() gin.HandlerFunc {
	return gin.BasicAuth(gin.Accounts{
		"admin": getAdminPassword(),
	})
}

// getAdminPassword returns the admin password from environment or default
func getAdminPassword() string {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin123" // Default password for development
	}
	return password
}

// ServeAdminDashboard serves the main admin dashboard
func (h *AdminHandler) ServeAdminDashboard(c *gin.Context) {
	// Get counts for dashboard stats
	var userCount, sourceCount, articleCount int64
	h.db.Model(&models.User{}).Count(&userCount)
	h.db.Model(&models.Source{}).Count(&sourceCount)
	h.db.Model(&models.Article{}).Count(&articleCount)

	// Get recent activity
	var recentArticles []models.Article
	h.db.Preload("SourceArticles.Source").
		Order("created_at DESC").
		Limit(5).
		Find(&recentArticles)

	html := h.generateAdminDashboardHTML(userCount, sourceCount, articleCount, recentArticles)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// ServeUsersPage serves the users management page
func (h *AdminHandler) ServeUsersPage(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit := 20
	offset := (page - 1) * limit

	var users []models.User
	var totalUsers int64

	h.db.Model(&models.User{}).Count(&totalUsers)
	h.db.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users)

	html := h.generateUsersPageHTML(users, page, limit, totalUsers)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// ServeSourcesPage serves the sources management page
func (h *AdminHandler) ServeSourcesPage(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit := 20
	offset := (page - 1) * limit

	var sources []models.Source
	var totalSources int64

	h.db.Model(&models.Source{}).Count(&totalSources)
	h.db.Order("quality_score DESC").
		Limit(limit).
		Offset(offset).
		Find(&sources)

	html := h.generateSourcesPageHTML(sources, page, limit, totalSources)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// ServeArticlesPage serves the articles management page
func (h *AdminHandler) ServeArticlesPage(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit := 20
	offset := (page - 1) * limit

	var articles []models.Article
	var totalArticles int64

	h.db.Model(&models.Article{}).Count(&totalArticles)
	h.db.Preload("SourceArticles.Source").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&articles)

	html := h.generateArticlesPageHTML(articles, page, limit, totalArticles)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// generateAdminDashboardHTML generates the main admin dashboard
func (h *AdminHandler) generateAdminDashboardHTML(userCount, sourceCount, articleCount int64, recentArticles []models.Article) string {
	return `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>open.news Admin</title>
    <link rel="stylesheet" href="/static/feed.css">
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css">
    <style>
        .admin-nav {
            background: #1e293b;
            padding: 1rem 0;
            margin-bottom: 2rem;
        }
        .admin-nav .nav-container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 0 1rem;
            display: flex;
            align-items: center;
            gap: 2rem;
        }
        .admin-nav .nav-brand {
            color: #f1f5f9;
            font-weight: 700;
            font-size: 1.25rem;
        }
        .admin-nav .nav-links {
            display: flex;
            gap: 1rem;
        }
        .admin-nav .nav-link {
            color: #cbd5e1;
            text-decoration: none;
            padding: 0.5rem 1rem;
            border-radius: 6px;
            transition: all 0.2s;
        }
        .admin-nav .nav-link:hover,
        .admin-nav .nav-link.active {
            background: #3b82f6;
            color: white;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }
        .stat-card {
            background: white;
            padding: 1.5rem;
            border-radius: 12px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            text-align: center;
        }
        .stat-number {
            font-size: 2.5rem;
            font-weight: 700;
            color: #3b82f6;
            margin-bottom: 0.5rem;
        }
        .stat-label {
            color: #64748b;
            font-weight: 500;
        }
        .recent-activity {
            background: white;
            padding: 1.5rem;
            border-radius: 12px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .activity-item {
            padding: 1rem 0;
            border-bottom: 1px solid #e2e8f0;
        }
        .activity-item:last-child {
            border-bottom: none;
        }
    </style>
</head>
<body>
    <nav class="admin-nav">
        <div class="nav-container">
            <div class="nav-brand">
                <i class="fas fa-shield-alt"></i> open.news Admin
            </div>
            <div class="nav-links">
                <a href="/admin" class="nav-link active">Dashboard</a>
                <a href="/admin/users" class="nav-link">Users</a>
                <a href="/admin/sources" class="nav-link">Sources</a>
                <a href="/admin/articles" class="nav-link">Articles</a>
                <a href="/" class="nav-link">← Back to Site</a>
            </div>
        </div>
    </nav>

    <div class="main-content">
        <h1>Admin Dashboard</h1>
        
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-number">` + strconv.FormatInt(userCount, 10) + `</div>
                <div class="stat-label">Users</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">` + strconv.FormatInt(sourceCount, 10) + `</div>
                <div class="stat-label">Sources</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">` + strconv.FormatInt(articleCount, 10) + `</div>
                <div class="stat-label">Articles</div>
            </div>
        </div>

        <div class="recent-activity">
            <h2>Recent Articles</h2>
            ` + h.generateRecentArticlesHTML(recentArticles) + `
        </div>
    </div>
</body>
</html>`
}

// generateRecentArticlesHTML generates HTML for recent articles
func (h *AdminHandler) generateRecentArticlesHTML(articles []models.Article) string {
	if len(articles) == 0 {
		return `<p>No articles found.</p>`
	}

	html := ""
	for _, article := range articles {
		sourceName := "Unknown Source"
		if len(article.SourceArticles) > 0 && article.SourceArticles[0].Source.ID != uuid.Nil {
			sourceName = article.SourceArticles[0].Source.DisplayName
		}

		html += `
        <div class="activity-item">
            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                <div>
                    <h4 style="margin: 0 0 0.5rem 0;">` + article.Title + `</h4>
                    <p style="margin: 0; color: #64748b; font-size: 0.875rem;">
                        by ` + sourceName + ` • ` + article.CreatedAt.Format("Jan 2, 3:04 PM") + `
                    </p>
                </div>
                <div style="background: #f1f5f9; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem;">
                    Score: ` + strconv.FormatFloat(article.QualityScore, 'f', 1, 64) + `
                </div>
            </div>
        </div>`
	}

	return html
}

// generateUsersPageHTML generates the users management page
func (h *AdminHandler) generateUsersPageHTML(users []models.User, page, limit int, total int64) string {
	html := h.generateAdminLayout("Users", `/admin/users`)
	
	html += `
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem;">
            <h1>Users (` + strconv.FormatInt(total, 10) + `)</h1>
        </div>

        <div style="background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
            <table style="width: 100%; border-collapse: collapse;">
                <thead style="background: #f8fafc;">
                    <tr>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Handle</th>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Display Name</th>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">DID</th>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Active</th>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Last Refresh</th>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Actions</th>
                    </tr>
                </thead>
                <tbody>`

	for _, user := range users {
		activeStatus := "❌"
		if user.IsActive {
			activeStatus = "✅"
		}

		lastRefresh := "Never"
		if !user.FollowsLastRefreshed.IsZero() {
			lastRefresh = user.FollowsLastRefreshed.Format("Jan 2, 15:04")
		}

		html += `
                    <tr style="border-bottom: 1px solid #f1f5f9;">
                        <td style="padding: 1rem;">@` + user.Handle + `</td>
                        <td style="padding: 1rem;">` + user.DisplayName + `</td>
                        <td style="padding: 1rem; font-family: monospace; font-size: 0.875rem;">` + user.BlueSkyDID[:20] + `...</td>
                        <td style="padding: 1rem;">` + activeStatus + `</td>
                        <td style="padding: 1rem;">` + lastRefresh + `</td>
                        <td style="padding: 1rem;">
                            <button onclick="refreshUserFollows('` + user.Handle + `')" 
                                    style="background: #3b82f6; color: white; border: none; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.875rem;">
                                🔄 Refresh
                            </button>
                        </td>
                    </tr>`
	}

	html += `
                </tbody>
            </table>
        </div>

        ` + h.generatePagination(page, limit, total, "/admin/users") + `
    </div>

    <script>
        function refreshUserFollows(userHandle) {
            const button = event.target;
            const originalText = button.innerHTML;
            
            // Show loading state
            button.innerHTML = '⏳ Refreshing...';
            button.disabled = true;
            
            fetch('/admin/refresh-follows/' + encodeURIComponent(userHandle), {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    button.innerHTML = '✅ Done';
                    button.style.background = '#10b981';
                    setTimeout(() => {
                        button.innerHTML = originalText;
                        button.style.background = '#3b82f6';
                        button.disabled = false;
                        // Reload the page to show updated refresh time
                        window.location.reload();
                    }, 2000);
                } else {
                    button.innerHTML = '❌ Error';
                    button.style.background = '#ef4444';
                    setTimeout(() => {
                        button.innerHTML = originalText;
                        button.style.background = '#3b82f6';
                        button.disabled = false;
                    }, 3000);
                    alert('Error: ' + (data.error || 'Unknown error'));
                }
            })
            .catch(error => {
                button.innerHTML = '❌ Error';
                button.style.background = '#ef4444';
                setTimeout(() => {
                    button.innerHTML = originalText;
                    button.style.background = '#3b82f6';
                    button.disabled = false;
                }, 3000);
                alert('Network error: ' + error.message);
            });
        }
    </script>
</body>
</html>`

	return html
}

// generateSourcesPageHTML generates the sources management page
func (h *AdminHandler) generateSourcesPageHTML(sources []models.Source, page, limit int, total int64) string {
	html := h.generateAdminLayout("Sources", `/admin/sources`)
	
	html += `
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem;">
            <h1>Sources (` + strconv.FormatInt(total, 10) + `)</h1>
        </div>

        <div style="background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
            <table style="width: 100%; border-collapse: collapse;">
                <thead style="background: #f8fafc;">
                    <tr>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Handle</th>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Display Name</th>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Quality Score</th>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Verified</th>
                        <th style="padding: 1rem; text-align: left; border-bottom: 1px solid #e2e8f0;">Created</th>
                    </tr>
                </thead>
                <tbody>`

	for _, source := range sources {
		verifiedStatus := "❌"
		if source.IsVerified {
			verifiedStatus = "✅"
		}

		qualityClass := "background: #fef2f2; color: #991b1b;" // Low
		if source.QualityScore >= 0.7 {
			qualityClass = "background: #f0fdf4; color: #166534;" // High
		} else if source.QualityScore >= 0.5 {
			qualityClass = "background: #fefce8; color: #a16207;" // Medium
		}

		html += `
                    <tr style="border-bottom: 1px solid #f1f5f9;">
                        <td style="padding: 1rem;">@` + source.Handle + `</td>
                        <td style="padding: 1rem;">` + source.DisplayName + `</td>
                        <td style="padding: 1rem;">
                            <span style="padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.875rem; ` + qualityClass + `">
                                ` + strconv.FormatFloat(source.QualityScore, 'f', 2, 64) + `
                            </span>
                        </td>
                        <td style="padding: 1rem;">` + verifiedStatus + `</td>
                        <td style="padding: 1rem;">` + source.CreatedAt.Format("Jan 2, 2006") + `</td>
                    </tr>`
	}

	html += `
                </tbody>
            </table>
        </div>

        ` + h.generatePagination(page, limit, total, "/admin/sources") + `
    </div>
</body>
</html>`

	return html
}

// generateArticlesPageHTML generates the articles management page
func (h *AdminHandler) generateArticlesPageHTML(articles []models.Article, page, limit int, total int64) string {
	html := h.generateAdminLayout("Articles", `/admin/articles`)
	
	html += `
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem;">
            <h1>Articles (` + strconv.FormatInt(total, 10) + `)</h1>
        </div>

        <div style="background: white; border-radius: 12px; padding: 1.5rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">`

	for _, article := range articles {
		sourceName := "Unknown Source"
		if len(article.SourceArticles) > 0 && article.SourceArticles[0].Source.ID != uuid.Nil {
			sourceName = article.SourceArticles[0].Source.DisplayName
		}

		qualityClass := "background: #fef2f2; color: #991b1b;" // Low
		if article.QualityScore >= 0.7 {
			qualityClass = "background: #f0fdf4; color: #166534;" // High
		} else if article.QualityScore >= 0.5 {
			qualityClass = "background: #fefce8; color: #a16207;" // Medium
		}

		html += `
            <div style="border-bottom: 1px solid #e2e8f0; padding: 1.5rem 0;">
                <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem;">
                    <div style="flex: 1;">
                        <h3 style="margin: 0 0 0.5rem 0;">
                            <a href="` + article.URL + `" target="_blank" style="color: #3b82f6; text-decoration: none;">
                                ` + article.Title + `
                            </a>
                        </h3>
                        <p style="margin: 0 0 0.5rem 0; color: #64748b; line-height: 1.5;">` + article.Description + `</p>
                        <div style="display: flex; align-items: center; gap: 1rem; font-size: 0.875rem; color: #64748b;">
                            <span>by ` + sourceName + `</span>
                            <span>•</span>
                            <span>` + article.CreatedAt.Format("Jan 2, 2006 3:04 PM") + `</span>
                            <span style="padding: 0.25rem 0.5rem; border-radius: 4px; ` + qualityClass + `">
                                Score: ` + strconv.FormatFloat(article.QualityScore, 'f', 1, 64) + `
                            </span>`

		// Add fetch status indicator
		if !article.IsReachable {
			html += `
                            <span style="padding: 0.25rem 0.5rem; border-radius: 4px; background: #fef2f2; color: #991b1b; border: 1px solid #fecaca;">
                                ❌ Unreachable
                            </span>`
		}

		html += `
                            <span>•</span>
                            <a href="/admin/articles/` + article.ID.String() + `" 
                               style="color: #3b82f6; text-decoration: none; padding: 0.25rem 0.5rem; background: #eff6ff; border-radius: 4px; border: 1px solid #dbeafe;">
                                🔍 Inspect
                            </a>
                        </div>
                    </div>`

		if article.ImageURL != "" {
			html += `
                    <img src="` + article.ImageURL + `" 
                         alt="Article image" 
                         style="width: 120px; height: 120px; object-fit: cover; border-radius: 8px; flex-shrink: 0;">`
		}

		html += `
                </div>
            </div>`
	}

	html += `
        </div>

        ` + h.generatePagination(page, limit, total, "/admin/articles") + `
    </div>
</body>
</html>`

	return html
}

// generateAdminLayout generates the common admin layout
func (h *AdminHandler) generateAdminLayout(title, activePath string) string {
	return `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + ` - open.news Admin</title>
    <link rel="stylesheet" href="/static/feed.css">
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css">
    <style>
        .admin-nav {
            background: #1e293b;
            padding: 1rem 0;
            margin-bottom: 2rem;
        }
        .admin-nav .nav-container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 0 1rem;
            display: flex;
            align-items: center;
            gap: 2rem;
        }
        .admin-nav .nav-brand {
            color: #f1f5f9;
            font-weight: 700;
            font-size: 1.25rem;
        }
        .admin-nav .nav-links {
            display: flex;
            gap: 1rem;
        }
        .admin-nav .nav-link {
            color: #cbd5e1;
            text-decoration: none;
            padding: 0.5rem 1rem;
            border-radius: 6px;
            transition: all 0.2s;
        }
        .admin-nav .nav-link:hover,
        .admin-nav .nav-link.active {
            background: #3b82f6;
            color: white;
        }
    </style>
</head>
<body>
    <nav class="admin-nav">
        <div class="nav-container">
            <div class="nav-brand">
                <i class="fas fa-shield-alt"></i> open.news Admin
            </div>
            <div class="nav-links">
                <a href="/admin" class="nav-link` + h.getActiveClass("/admin", activePath) + `">Dashboard</a>
                <a href="/admin/users" class="nav-link` + h.getActiveClass("/admin/users", activePath) + `">Users</a>
                <a href="/admin/sources" class="nav-link` + h.getActiveClass("/admin/sources", activePath) + `">Sources</a>
                <a href="/admin/articles" class="nav-link` + h.getActiveClass("/admin/articles", activePath) + `">Articles</a>
                <a href="/" class="nav-link">← Back to Site</a>
            </div>
        </div>
    </nav>

    <div class="main-content">`
}

// getActiveClass returns " active" if the path matches
func (h *AdminHandler) getActiveClass(path, activePath string) string {
	if path == activePath {
		return " active"
	}
	return ""
}

// generatePagination generates pagination controls
func (h *AdminHandler) generatePagination(currentPage, limit int, total int64, basePath string) string {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	
	if totalPages <= 1 {
		return ""
	}

	html := `
    <div style="display: flex; justify-content: center; gap: 0.5rem; margin-top: 2rem; padding-top: 2rem; border-top: 1px solid #e2e8f0;">
        ` + h.getPaginationButton(basePath, currentPage-1, "Previous", currentPage <= 1) + `
        <span style="padding: 0.5rem 1rem; background: #3b82f6; color: white; border-radius: 6px;">
            Page ` + strconv.Itoa(currentPage) + ` of ` + strconv.Itoa(totalPages) + `
        </span>
        ` + h.getPaginationButton(basePath, currentPage+1, "Next", currentPage >= totalPages) + `
    </div>`

	return html
}

// getPaginationButton generates a pagination button
func (h *AdminHandler) getPaginationButton(basePath string, page int, text string, disabled bool) string {
	if disabled {
		return `<span style="padding: 0.5rem 1rem; background: #f1f5f9; color: #94a3b8; border-radius: 6px;">` + text + `</span>`
	}
	return `<a href="` + basePath + `?page=` + strconv.Itoa(page) + `" style="padding: 0.5rem 1rem; background: white; color: #3b82f6; border: 1px solid #e2e8f0; border-radius: 6px; text-decoration: none; transition: all 0.2s;" onmouseover="this.style.background='#f1f5f9'" onmouseout="this.style.background='white'">` + text + `</a>`
}

// RefreshUserFollows handles manual refresh of user follows
func (h *AdminHandler) RefreshUserFollows(c *gin.Context) {
	userIdentifier := c.Param("user")
	if userIdentifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user identifier (handle or DID) is required"})
		return
	}

	// Force refresh config (ignore time limits)
	config := services.RefreshConfig{
		RefreshInterval: 0, // Force immediate refresh
		BatchSize:       10,
		RateLimit:       100 * time.Millisecond,
	}

	// Find the user by DID or handle
	var user models.User
	var err error
	if len(userIdentifier) > 20 && (userIdentifier[:8] == "did:plc:" || userIdentifier[:8] == "did:web:") {
		// Looks like a DID
		err = h.db.Where("blue_sky_d_id = ?", userIdentifier).First(&user).Error
	} else {
		// Assume it's a handle (with or without @)
		handle := userIdentifier
		if handle[0] == '@' {
			handle = handle[1:] // Remove @ prefix if present
		}
		err = h.db.Where("handle = ?", handle).First(&user).Error
	}
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Import follows
	if err := h.userFollowsService.ImportUserFollows(&user, config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh follows: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Successfully refreshed follows for user " + user.Handle,
	})
}

// RefreshAllUserFollows handles manual refresh of all users' follows
func (h *AdminHandler) RefreshAllUserFollows(c *gin.Context) {
	// Force refresh config (ignore time limits)
	config := services.RefreshConfig{
		RefreshInterval: 0, // Force immediate refresh for all users
		BatchSize:       50, // Process more users at once for manual refresh
		RateLimit:       100 * time.Millisecond,
	}

	if err := h.userFollowsService.RefreshBatch(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh follows: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Successfully triggered refresh for all users",
	})
}

// ServeArticleInspection serves the detailed article inspection page
func (h *AdminHandler) ServeArticleInspection(c *gin.Context) {
	articleID := c.Param("id")
	
	// Parse UUID
	id, err := uuid.Parse(articleID)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid article ID")
		return
	}

	// Get article with all related data
	var article models.Article
	result := h.db.Preload("SourceArticles.Source").
		Preload("Facts").
		First(&article, id)
	
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.String(http.StatusNotFound, "Article not found")
			return
		}
		c.String(http.StatusInternalServerError, "Database error: "+result.Error.Error())
		return
	}

	html := h.generateArticleInspectionHTML(article)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// generateArticleInspectionHTML generates the detailed article inspection page
func (h *AdminHandler) generateArticleInspectionHTML(article models.Article) string {
	html := h.generateAdminLayout("Article Inspection", "/admin/articles")
	
	// Determine quality score styling
	qualityClass := "background: #fef2f2; color: #991b1b; border: 1px solid #fecaca;" // Low
	qualityIcon := "⚠️"
	if article.QualityScore >= 0.7 {
		qualityClass = "background: #f0fdf4; color: #166534; border: 1px solid #bbf7d0;" // High
		qualityIcon = "✅"
	} else if article.QualityScore >= 0.5 {
		qualityClass = "background: #fefce8; color: #a16207; border: 1px solid #fde68a;" // Medium
		qualityIcon = "⚡"
	}

	// Get source information
	sourceName := "Unknown Source"
	sourceHandle := "N/A"
	sourceDID := "N/A"
	postURI := "N/A"
	if len(article.SourceArticles) > 0 {
		sourceArticle := article.SourceArticles[0]
		if sourceArticle.Source.ID != uuid.Nil {
			sourceName = sourceArticle.Source.DisplayName
			sourceHandle = sourceArticle.Source.Handle
			sourceDID = sourceArticle.Source.BlueSkyDID
		}
		postURI = sourceArticle.PostURI
	}

	html += `
        <div style="margin-bottom: 1.5rem;">
            <a href="/admin/articles" style="color: #3b82f6; text-decoration: none; font-size: 0.875rem;">
                ← Back to Articles
            </a>
        </div>

        <div style="background: white; border-radius: 12px; padding: 2rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
            <div style="border-bottom: 1px solid #e2e8f0; padding-bottom: 1.5rem; margin-bottom: 1.5rem;">
                <h1 style="margin: 0 0 1rem 0; color: #1e293b; font-size: 1.5rem;">Article Inspection</h1>
                <div style="padding: 1rem; border-radius: 8px; ` + qualityClass + `">
                    <strong>` + qualityIcon + ` Quality Score: ` + strconv.FormatFloat(article.QualityScore, 'f', 3, 64) + `</strong>
                </div>
            </div>

            <!-- Article Content -->
            <div style="margin-bottom: 2rem;">
                <h2 style="color: #1e293b; margin-bottom: 1rem; border-bottom: 2px solid #e2e8f0; padding-bottom: 0.5rem;">Content</h2>
                <div style="display: grid; gap: 1rem;">
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Title:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + article.Title + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Description:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0; line-height: 1.5;">` + article.Description + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">URL:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">
                            <a href="` + article.URL + `" target="_blank" style="color: #3b82f6; text-decoration: none;">` + article.URL + `</a>
                        </div>
                    </div>`
	
	if article.ImageURL != "" {
		html += `
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Image:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">
                            <a href="` + article.ImageURL + `" target="_blank" style="color: #3b82f6; text-decoration: none;">` + article.ImageURL + `</a><br>
                            <img src="` + article.ImageURL + `" alt="Article image" style="max-width: 200px; max-height: 200px; object-fit: cover; border-radius: 6px; margin-top: 0.5rem;">
                        </div>
                    </div>`
	}

	html += `
                </div>
            </div>

            <!-- Metadata -->
            <div style="margin-bottom: 2rem;">
                <h2 style="color: #1e293b; margin-bottom: 1rem; border-bottom: 2px solid #e2e8f0; padding-bottom: 0.5rem;">Metadata</h2>
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Word Count:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + strconv.Itoa(article.WordCount) + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Reading Time:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + strconv.Itoa(article.ReadingTime) + ` min</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Language:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + article.Language + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Site Name:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + article.SiteName + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Author:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + article.Author + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Created:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + article.CreatedAt.Format("Jan 2, 2006 3:04:05 PM") + `</div>
                    </div>
                </div>
            </div>

            <!-- Fetch Status -->
            <div style="margin-bottom: 2rem;">
                <h2 style="color: #1e293b; margin-bottom: 1rem; border-bottom: 2px solid #e2e8f0; padding-bottom: 0.5rem;">Fetch Status</h2>
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">`

	// Fetch status styling
	reachableClass := "background: #f0fdf4; color: #166534; border: 1px solid #bbf7d0;" // Green for reachable
	reachableIcon := "✅"
	reachableText := "Reachable"
	
	if !article.IsReachable {
		reachableClass = "background: #fef2f2; color: #991b1b; border: 1px solid #fecaca;" // Red for unreachable
		reachableIcon = "❌"
		reachableText = "Unreachable"
	}

	html += `
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Status:</label>
                        <div style="padding: 0.75rem; border-radius: 6px; ` + reachableClass + `">` + reachableIcon + ` ` + reachableText + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Fetch Retries:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + strconv.Itoa(article.FetchRetries) + `</div>
                    </div>`

	if article.LastFetchAt != nil {
		html += `
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Last Fetch Attempt:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + article.LastFetchAt.Format("Jan 2, 2006 3:04:05 PM") + `</div>
                    </div>`
	}

	if article.LastFetchError != nil {
		html += `
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Last Error Time:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + article.LastFetchError.Format("Jan 2, 2006 3:04:05 PM") + `</div>
                    </div>`
	}

	if article.FetchError != "" {
		html += `
                    <div style="grid-column: 1 / -1;">
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Last Error Message:</label>
                        <div style="padding: 0.75rem; background: #fef2f2; border-radius: 6px; border: 1px solid #fecaca; font-family: monospace; font-size: 0.875rem; color: #991b1b;">` + article.FetchError + `</div>
                    </div>`
	}

	html += `
                </div>
            </div>

            <!-- Source Information -->
            <div style="margin-bottom: 2rem;">
                <h2 style="color: #1e293b; margin-bottom: 1rem; border-bottom: 2px solid #e2e8f0; padding-bottom: 0.5rem;">Source Information</h2>
                <div style="display: grid; gap: 1rem;">
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Source Name:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + sourceName + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Bluesky Handle:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">` + sourceHandle + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Bluesky DID:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0; font-family: monospace; font-size: 0.875rem;">` + sourceDID + `</div>
                    </div>
                    <div>
                        <label style="font-weight: 600; color: #374151; display: block; margin-bottom: 0.5rem;">Post URI:</label>
                        <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0; font-family: monospace; font-size: 0.875rem; word-break: break-all;">` + postURI + `</div>
                    </div>
                </div>
            </div>`

	// Article Facts section
	if len(article.Facts) > 0 {
		html += `
            <div style="margin-bottom: 2rem;">
                <h2 style="color: #1e293b; margin-bottom: 1rem; border-bottom: 2px solid #e2e8f0; padding-bottom: 0.5rem;">Article Facts</h2>
                <div style="display: grid; gap: 0.5rem;">`

		for _, fact := range article.Facts {
			html += `
                    <div style="padding: 0.75rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0; display: flex; justify-content: space-between; align-items: center;">
                        <span style="font-weight: 500;">` + fact.FactType + `: ` + fact.FactText + `</span>
                        <span style="font-family: monospace; background: #e2e8f0; padding: 0.25rem 0.5rem; border-radius: 4px;">Confidence: ` + strconv.FormatFloat(fact.Confidence, 'f', 3, 64) + `</span>
                    </div>`
		}

		html += `
                </div>
            </div>`
	}

	// Raw JSON section for debugging
	html += `
            <div style="margin-bottom: 2rem;">
                <h2 style="color: #1e293b; margin-bottom: 1rem; border-bottom: 2px solid #e2e8f0; padding-bottom: 0.5rem;">Raw Data (JSON)</h2>
                <details style="margin-bottom: 1rem;">
                    <summary style="cursor: pointer; padding: 0.5rem; background: #f8fafc; border-radius: 6px; font-weight: 500;">Article JSON</summary>
                    <pre style="background: #1e293b; color: #e2e8f0; padding: 1rem; border-radius: 6px; overflow-x: auto; margin-top: 0.5rem; font-size: 0.875rem; line-height: 1.4;">`

	// Create a simplified JSON representation for display
	html += `{
  "id": "` + article.ID.String() + `",
  "url": "` + article.URL + `",
  "title": "` + article.Title + `",
  "description": "` + article.Description + `",
  "image_url": "` + article.ImageURL + `",
  "site_name": "` + article.SiteName + `",
  "author": "` + article.Author + `",
  "language": "` + article.Language + `",
  "word_count": ` + strconv.Itoa(article.WordCount) + `,
  "reading_time": ` + strconv.Itoa(article.ReadingTime) + `,
  "quality_score": ` + strconv.FormatFloat(article.QualityScore, 'f', 6, 64) + `,
  "created_at": "` + article.CreatedAt.Format(time.RFC3339) + `",
  "updated_at": "` + article.UpdatedAt.Format(time.RFC3339) + `"
}`

	html += `</pre>
                </details>
            </div>
        </div>
    </div>
</body>
</html>`

	return html
}
