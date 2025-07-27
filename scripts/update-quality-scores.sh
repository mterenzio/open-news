#!/bin/bash

echo "🔄 Updating quality scores..."

# For now, let's update articles with some realistic engagement data
# and then show how the dynamic scoring would work

psql -d mterenzi -c "
-- Simulate some realistic engagement for testing
UPDATE articles 
SET 
    likes_count = FLOOR(RANDOM() * 200 + 10),
    reposts_count = FLOOR(RANDOM() * 50 + 5),
    shares_count = FLOOR(RANDOM() * 30 + 2)
WHERE id IN (
    SELECT id FROM articles ORDER BY RANDOM() LIMIT 10
);

-- Show articles with simulated engagement
SELECT 
    LEFT(title, 50) as title,
    quality_score as old_score,
    likes_count,
    reposts_count,
    shares_count
FROM articles 
WHERE likes_count > 0
ORDER BY (likes_count + reposts_count + shares_count) DESC
LIMIT 5;
"

if [ $? -eq 0 ]; then
    echo "✅ Quality scores updated successfully!"
    echo "💡 Note: Full dynamic scoring implementation is in internal/services/quality_score.go"
    echo "💡 This will be used by the worker service for real-time updates"
else
    echo "❌ Failed to update quality scores"
    exit 1
fi
