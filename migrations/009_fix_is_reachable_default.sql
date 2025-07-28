-- Fix default value for is_reachable column
-- Should default to false, not true
ALTER TABLE articles ALTER COLUMN is_reachable SET DEFAULT false;
