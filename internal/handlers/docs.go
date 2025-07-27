package handlers

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/russross/blackfriday/v2"
)

type DocsHandler struct{}

func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

// ServeMarkdownAsHTML serves Markdown files as HTML with consistent styling
func (h *DocsHandler) ServeMarkdownAsHTML(c *gin.Context) {
	// Get the requested document name from the URL
	docName := c.Param("doc")
	if docName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document name required"})
		return
	}

	// Security: Only allow specific documentation files
	allowedDocs := map[string]string{
		"README":                "README.md",
		"DEVELOPMENT":           "DEVELOPMENT.md",
		"TESTING":               "TESTING.md",
		"PRODUCTION_DEPLOYMENT": "PRODUCTION_DEPLOYMENT.md",
		"BLUESKY_FEEDS":         "BLUESKY_FEEDS.md",
		"QUICK_DEPLOY":          "QUICK_DEPLOY.md",
		"STATUS":                "STATUS.md",
	}

	fileName, exists := allowedDocs[docName]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Read the Markdown file
	filePath := filepath.Join(".", fileName)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Convert Markdown to HTML
	extensions := blackfriday.CommonExtensions | blackfriday.AutoHeadingIDs
	renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
		Flags: blackfriday.CommonHTMLFlags,
	})
	htmlContent := blackfriday.Run(content, blackfriday.WithRenderer(renderer), blackfriday.WithExtensions(extensions))

	// Get a human-readable title
	title := getDocumentTitle(docName)

	// Serve the HTML with consistent styling
	html := h.wrapWithTheme(string(htmlContent), title, docName)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// getDocumentTitle returns a human-readable title for the document
func getDocumentTitle(docName string) string {
	titles := map[string]string{
		"README":                "Project Overview",
		"DEVELOPMENT":           "Development Setup",
		"TESTING":               "Testing Guide",
		"PRODUCTION_DEPLOYMENT": "Production Deployment",
		"BLUESKY_FEEDS":         "Bluesky Integration",
		"QUICK_DEPLOY":          "Quick Deploy Guide",
		"STATUS":                "Project Status",
	}
	
	if title, exists := titles[docName]; exists {
		return title
	}
	return strings.ReplaceAll(docName, "_", " ")
}

// wrapWithTheme wraps the HTML content with consistent styling
func (h *DocsHandler) wrapWithTheme(content, title, docName string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + ` - Open News</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            line-height: 1.6;
            color: #333;
            background: #f8f9fa;
            padding: 20px;
        }
        
        .container {
            max-width: 1000px;
            margin: 0 auto;
        }
        
        .header {
            background: linear-gradient(135deg, #2563eb 0%, #3b82f6 100%);
            color: white;
            padding: 2rem;
            margin-bottom: 2rem;
            border-radius: 12px;
            text-align: center;
            box-shadow: 0 4px 20px rgba(37, 99, 235, 0.3);
        }
        
        .header h1 {
            font-size: 2.2rem;
            margin-bottom: 0.5rem;
            font-weight: 700;
        }
        
        .header .breadcrumb {
            font-size: 1rem;
            opacity: 0.9;
        }
        
        .header .breadcrumb a {
            color: white;
            text-decoration: none;
            opacity: 0.8;
        }
        
        .header .breadcrumb a:hover {
            opacity: 1;
            text-decoration: underline;
        }
        
        .content {
            background: white;
            padding: 3rem;
            border-radius: 12px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            border: 1px solid #e5e7eb;
        }
        
        .content h1, .content h2, .content h3, .content h4, .content h5, .content h6 {
            color: #1f2937;
            margin-top: 2rem;
            margin-bottom: 1rem;
            font-weight: 600;
        }
        
        .content h1 {
            font-size: 2rem;
            border-bottom: 2px solid #e5e7eb;
            padding-bottom: 0.5rem;
            margin-top: 0;
        }
        
        .content h2 {
            font-size: 1.5rem;
            color: #2563eb;
        }
        
        .content h3 {
            font-size: 1.25rem;
        }
        
        .content p {
            margin-bottom: 1rem;
            color: #374151;
        }
        
        .content ul, .content ol {
            margin-bottom: 1rem;
            padding-left: 2rem;
        }
        
        .content li {
            margin-bottom: 0.5rem;
            color: #374151;
        }
        
        .content pre {
            background: #f3f4f6;
            border: 1px solid #d1d5db;
            border-radius: 8px;
            padding: 1.5rem;
            overflow-x: auto;
            margin-bottom: 1.5rem;
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
            font-size: 0.9rem;
        }
        
        .content code {
            background: #f3f4f6;
            padding: 0.2rem 0.4rem;
            border-radius: 4px;
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
            font-size: 0.9rem;
            color: #2563eb;
        }
        
        .content pre code {
            background: none;
            padding: 0;
            color: #374151;
        }
        
        .content blockquote {
            border-left: 4px solid #2563eb;
            padding-left: 1rem;
            margin: 1.5rem 0;
            color: #6b7280;
            font-style: italic;
        }
        
        .content table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 1.5rem;
        }
        
        .content th, .content td {
            border: 1px solid #d1d5db;
            padding: 0.75rem;
            text-align: left;
        }
        
        .content th {
            background: #f9fafb;
            font-weight: 600;
            color: #374151;
        }
        
        .content a {
            color: #2563eb;
            text-decoration: none;
        }
        
        .content a:hover {
            text-decoration: underline;
        }
        
        .content img {
            max-width: 100%;
            height: auto;
            border-radius: 8px;
            margin: 1rem 0;
        }
        
        .back-to-home {
            margin-top: 2rem;
            text-align: center;
        }
        
        .back-to-home a {
            display: inline-block;
            background: #2563eb;
            color: white;
            padding: 0.75rem 1.5rem;
            border-radius: 8px;
            text-decoration: none;
            font-weight: 500;
            transition: background 0.2s;
        }
        
        .back-to-home a:hover {
            background: #1d4ed8;
        }
        
        @media (max-width: 768px) {
            body {
                padding: 10px;
            }
            
            .header {
                padding: 1.5rem 1rem;
            }
            
            .header h1 {
                font-size: 1.8rem;
            }
            
            .content {
                padding: 2rem 1.5rem;
            }
            
            .content h1 {
                font-size: 1.6rem;
            }
            
            .content h2 {
                font-size: 1.3rem;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>` + title + `</h1>
            <div class="breadcrumb">
                <a href="/">🏠 Home</a> / 
                <a href="/">📚 Documentation</a> / 
                ` + title + `
            </div>
        </div>
        
        <div class="content">
            ` + content + `
        </div>
        
        <div class="back-to-home">
            <a href="/">← Back to Developer Dashboard</a>
        </div>
    </div>
</body>
</html>`
}
