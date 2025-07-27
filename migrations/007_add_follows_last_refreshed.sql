-- Migration: Add follows_last_refreshed column to users table
-- This field tracks when we last imported a user's follows from Bluesky

ALTER TABLE users ADD COLUMN follows_last_refreshed TIMESTAMP NULL;

-- Add index for performance when querying users who need follow refresh
CREATE INDEX idx_users_follows_last_refreshed ON users(follows_last_refreshed);
