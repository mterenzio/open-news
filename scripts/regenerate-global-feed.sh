#!/bin/bash

# Regenerate Global Feed Script
# This script populates the feed_items table for the global feed

echo "🔄 Regenerating global feed..."

# Check if psql is available and database is accessible
if ! command -v psql &> /dev/null; then
    echo "❌ psql command not found. Please install PostgreSQL client."
    exit 1
fi

# Clear existing feed items for global feed and regenerate
psql -d mterenzi -c "
-- Clear existing feed items for global feed
DELETE FROM feed_items 
WHERE feed_id IN (
    SELECT id FROM feeds WHERE feed_type = 'global' AND name = 'Top Stories'
);

-- Regenerate feed items
WITH global_feed AS (
  SELECT id FROM feeds WHERE feed_type = 'global' AND name = 'Top Stories' LIMIT 1
),
-- Get top articles from last 7 days with quality scores > 0
top_articles AS (
  SELECT 
    id,
    quality_score,
    trending_score,
    ROW_NUMBER() OVER (ORDER BY quality_score DESC, trending_score DESC, created_at DESC) as position
  FROM articles 
  WHERE created_at > NOW() - INTERVAL '7 days' 
    AND quality_score > 0
  LIMIT 100
)
-- Insert new feed items
INSERT INTO feed_items (
  id, 
  feed_id, 
  article_id, 
  position, 
  score, 
  relevance, 
  added_at, 
  created_at, 
  updated_at
)
SELECT 
  gen_random_uuid(),
  gf.id,
  ta.id,
  ta.position,
  -- Score calculation: quality + trending bonus + position bonus
  ta.quality_score + (ta.trending_score * 0.3) + ((101 - ta.position) * 0.01) as score,
  ta.quality_score,
  NOW(),
  NOW(),
  NOW()
FROM global_feed gf
CROSS JOIN top_articles ta;

-- Update feed timestamp
UPDATE feeds 
SET updated_at = NOW() 
WHERE feed_type = 'global' AND name = 'Top Stories';

-- Show results
SELECT 
  'Global feed regenerated with ' || COUNT(*) || ' items' as result
FROM feed_items fi
JOIN feeds f ON fi.feed_id = f.id
WHERE f.feed_type = 'global' AND f.name = 'Top Stories';
"

if [ $? -eq 0 ]; then
    echo "✅ Global feed regeneration completed successfully!"
else
    echo "❌ Failed to regenerate global feed"
    exit 1
fi
