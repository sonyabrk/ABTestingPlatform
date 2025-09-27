ALTER TABLE users 
ADD CONSTRAINT fk_users_experiments 
FOREIGN KEY (experiment_id) REFERENCES experiments(id) ON DELETE CASCADE;

ALTER TABLE results 
ADD CONSTRAINT fk_results_users 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;