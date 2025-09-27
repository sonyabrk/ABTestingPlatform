ALTER TABLE users 
DROP CONSTRAINT IF EXISTS fk_users_experiments;

ALTER TABLE results 
DROP CONSTRAINT IF EXISTS fk_results_users;