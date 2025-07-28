-- Fix unique constraint on source_articles to allow multiple articles per post
-- This migration removes the old unique constraint on post_uri alone
-- and adds a new composite unique constraint on (post_uri, article_id)

-- Drop the old unique index
DROP INDEX IF EXISTS idx_source_articles_post_uri;

-- Create new composite unique index
CREATE UNIQUE INDEX idx_source_articles_unique ON source_articles (post_uri, article_id);
