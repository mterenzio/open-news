-- Add fetch status tracking fields to articles table
ALTER TABLE articles 
ADD COLUMN is_reachable BOOLEAN DEFAULT true,
ADD COLUMN fetch_error TEXT,
ADD COLUMN fetch_retries INTEGER DEFAULT 0,
ADD COLUMN last_fetch_error TIMESTAMP;

-- Create an index on is_reachable for filtering feeds
CREATE INDEX idx_articles_is_reachable ON articles(is_reachable);

-- Create an index on fetch_retries for retry logic
CREATE INDEX idx_articles_fetch_retries ON articles(fetch_retries);
