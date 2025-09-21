ALTER TABLE experiments 
DROP CONSTRAINT IF EXISTS experiments_user_percent_check;

ALTER TABLE experiments 
ALTER COLUMN user_percent TYPE INTEGER
USING ROUND(user_percent)::INTEGER;

ALTER TABLE experiments 
ADD CONSTRAINT experiments_user_percent_check 
CHECK (user_percent > 0 AND user_percent <= 100);