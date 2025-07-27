package services

import (
	"log"
	"math"
	"open-news/internal/models"
	"time"

	"gorm.io/gorm"
)

// QualityScoreService handles dynamic quality score calculation
type QualityScoreService struct {
	db *gorm.DB
}

// NewQualityScoreService creates a new quality score service
func NewQualityScoreService(db *gorm.DB) *QualityScoreService {
	return &QualityScoreService{db: db}
}

// UpdateAllQualityScores recalculates quality scores for all articles
func (qs *QualityScoreService) UpdateAllQualityScores() error {
	log.Println("🔄 Starting quality score updates...")

	// First update source quality scores
	if err := qs.updateSourceQualityScores(); err != nil {
		return err
	}

	// Then update article quality scores
	if err := qs.updateArticleQualityScores(); err != nil {
		return err
	}

	// Finally update trending scores
	if err := qs.updateTrendingScores(); err != nil {
		return err
	}

	log.Println("✅ Quality score updates completed")
	return nil
}

// updateSourceQualityScores calculates quality scores for sources based on their articles
func (qs *QualityScoreService) updateSourceQualityScores() error {
	log.Println("📊 Updating source quality scores...")

	var sources []models.Source
	if err := qs.db.Find(&sources).Error; err != nil {
		return err
	}

	for _, source := range sources {
		score := qs.calculateSourceQualityScore(source.ID.String())
		
		if err := qs.db.Model(&source).Update("quality_score", score).Error; err != nil {
			log.Printf("Failed to update source %s quality score: %v", source.Handle, err)
			continue
		}
	}

	return nil
}

// calculateSourceQualityScore calculates quality score for a source
func (qs *QualityScoreService) calculateSourceQualityScore(sourceID string) float64 {
	// Get source's articles and their engagement
	var sourceArticles []models.SourceArticle
	qs.db.Preload("Article").Where("source_id = ?", sourceID).Find(&sourceArticles)

	if len(sourceArticles) == 0 {
		return 0.5 // Default score for new sources
	}

	var totalEngagement float64
	var totalShares int
	var validArticles int

	for _, sa := range sourceArticles {
		if sa.Article.ID.String() == "" {
			continue
		}

		// Calculate engagement rate
		totalEngagement += float64(sa.LikesCount + sa.RepostsCount + sa.RepliesCount)
		totalShares += sa.LikesCount + sa.RepostsCount
		validArticles++
	}

	if validArticles == 0 {
		return 0.5
	}

	// Base score from average engagement
	avgEngagement := totalEngagement / float64(validArticles)
	baseScore := math.Min(0.5 + (avgEngagement/1000.0), 1.0) // Cap at 1.0

	// Bonus for consistency (more articles = more reliable)
	consistencyBonus := math.Min(float64(validArticles)/100.0, 0.2)

	// Bonus for recent activity
	recentActivityBonus := qs.calculateRecentActivityBonus(sourceID)

	finalScore := baseScore + consistencyBonus + recentActivityBonus
	return math.Min(finalScore, 1.0) // Cap at 1.0
}

// calculateRecentActivityBonus gives bonus for sources that have been active recently
func (qs *QualityScoreService) calculateRecentActivityBonus(sourceID string) float64 {
	var count int64
	qs.db.Model(&models.SourceArticle{}).
		Where("source_id = ? AND created_at > ?", sourceID, time.Now().AddDate(0, 0, -7)).
		Count(&count)

	// Bonus up to 0.1 for recent activity
	return math.Min(float64(count)/50.0, 0.1)
}

// updateArticleQualityScores calculates quality scores for articles
func (qs *QualityScoreService) updateArticleQualityScores() error {
	log.Println("📰 Updating article quality scores...")

	// Get articles that need score updates (all articles for now)
	var articles []models.Article
	if err := qs.db.Preload("SourceArticles.Source").Find(&articles).Error; err != nil {
		return err
	}

	for _, article := range articles {
		score := qs.calculateArticleQualityScore(article)
		
		if err := qs.db.Model(&article).Update("quality_score", score).Error; err != nil {
			log.Printf("Failed to update article %s quality score: %v", article.URL, err)
			continue
		}
	}

	return nil
}

// calculateArticleQualityScore calculates quality score for an article
func (qs *QualityScoreService) calculateArticleQualityScore(article models.Article) float64 {
	var score float64 = 0.5 // Base score

	// 1. Source quality contribution (40% weight)
	if len(article.SourceArticles) > 0 {
		var avgSourceQuality float64
		for _, sa := range article.SourceArticles {
			avgSourceQuality += sa.Source.QualityScore
		}
		avgSourceQuality /= float64(len(article.SourceArticles))
		score += avgSourceQuality * 0.4
	}

	// 2. Engagement metrics (30% weight)
	totalEngagement := article.LikesCount + article.RepostsCount + article.SharesCount
	engagementScore := math.Min(float64(totalEngagement)/500.0, 0.3) // Cap at 0.3
	score += engagementScore

	// 3. Content quality indicators (20% weight)
	contentScore := qs.calculateContentQualityScore(article)
	score += contentScore * 0.2

	// 4. Domain reputation (10% weight)
	domainScore := qs.calculateDomainScore(article.SiteName)
	score += domainScore * 0.1

	return math.Min(score, 1.0) // Cap at 1.0
}

// calculateContentQualityScore evaluates content quality
func (qs *QualityScoreService) calculateContentQualityScore(article models.Article) float64 {
	var score float64 = 0.5

	// Word count bonus (articles with good length get bonus)
	if article.WordCount >= 300 && article.WordCount <= 3000 {
		score += 0.2
	} else if article.WordCount >= 150 {
		score += 0.1
	}

	// Title and description quality
	if len(article.Title) > 10 && len(article.Title) < 200 {
		score += 0.1
	}
	if len(article.Description) > 50 {
		score += 0.1
	}

	// Has media (image)
	if article.ImageURL != "" {
		score += 0.1
	}

	return math.Min(score, 1.0)
}

// calculateDomainScore gives reputation scores to known domains
func (qs *QualityScoreService) calculateDomainScore(siteName string) float64 {
	// High-quality news sources
	highQualitySources := map[string]float64{
		"Reuters":     1.0,
		"BBC News":    0.95,
		"The Guardian": 0.9,
		"Nature":      0.98,
		"arXiv":       0.9,
		"The New York Times": 0.92,
		"The Washington Post": 0.9,
		"Associated Press": 0.95,
	}

	// Medium-quality sources
	mediumQualitySources := map[string]float64{
		"TechCrunch":     0.8,
		"WIRED":          0.85,
		"The Economist":  0.88,
		"CNN":            0.75,
		"Forbes":         0.7,
		"Bloomberg":      0.85,
	}

	if score, exists := highQualitySources[siteName]; exists {
		return score
	}

	if score, exists := mediumQualitySources[siteName]; exists {
		return score
	}

	// Default score for unknown domains
	return 0.5
}

// updateTrendingScores calculates trending scores based on recent engagement
func (qs *QualityScoreService) updateTrendingScores() error {
	log.Println("📈 Updating trending scores...")

	// Get articles from the last 48 hours
	cutoff := time.Now().AddDate(0, 0, -2)
	var articles []models.Article
	if err := qs.db.Where("created_at > ?", cutoff).Find(&articles).Error; err != nil {
		return err
	}

	for _, article := range articles {
		trendingScore := qs.calculateTrendingScore(article)
		
		if err := qs.db.Model(&article).Update("trending_score", trendingScore).Error; err != nil {
			log.Printf("Failed to update article %s trending score: %v", article.URL, err)
			continue
		}
	}

	return nil
}

// calculateTrendingScore calculates how trending an article is
func (qs *QualityScoreService) calculateTrendingScore(article models.Article) float64 {
	now := time.Now()
	hoursSinceCreated := now.Sub(article.CreatedAt).Hours()

	// Decay factor: articles lose trending value over time
	decayFactor := math.Exp(-hoursSinceCreated / 24.0) // Half-life of 24 hours

	// Engagement velocity (engagement per hour)
	totalEngagement := float64(article.LikesCount + article.RepostsCount + article.SharesCount)
	velocity := totalEngagement / math.Max(hoursSinceCreated, 1.0)

	// Trending score based on velocity and decay
	trendingScore := velocity * decayFactor / 10.0 // Scale down

	return math.Min(trendingScore, 1.0)
}

// UpdateSingleArticleScore updates quality score for a specific article
func (qs *QualityScoreService) UpdateSingleArticleScore(articleID string) error {
	var article models.Article
	if err := qs.db.Preload("SourceArticles.Source").Where("id = ?", articleID).First(&article).Error; err != nil {
		return err
	}

	qualityScore := qs.calculateArticleQualityScore(article)
	trendingScore := qs.calculateTrendingScore(article)

	return qs.db.Model(&article).Updates(map[string]interface{}{
		"quality_score":  qualityScore,
		"trending_score": trendingScore,
	}).Error
}
