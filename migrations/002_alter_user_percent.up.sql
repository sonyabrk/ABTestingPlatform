ALTER TABLE experiments 
DROP CONSTRAINT IF EXISTS experiments_user_percent_check;

ALTER TABLE experiments 
ALTER COLUMN user_percent TYPE NUMERIC(5,2)
USING user_percent::NUMERIC(5,2);

ALTER TABLE experiments 
ADD CONSTRAINT experiments_user_percent_check 
CHECK (user_percent > 0.1 AND user_percent <= 100.0);