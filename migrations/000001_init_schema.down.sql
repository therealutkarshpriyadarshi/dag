-- Drop triggers
DROP TRIGGER IF EXISTS update_task_instances_updated_at ON task_instances;
DROP TRIGGER IF EXISTS update_dag_runs_updated_at ON dag_runs;
DROP TRIGGER IF EXISTS update_dags_updated_at ON dags;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS task_logs;
DROP TABLE IF EXISTS state_history;
DROP TABLE IF EXISTS task_instances;
DROP TABLE IF EXISTS dag_runs;
DROP TABLE IF EXISTS dags;

-- Drop extension
DROP EXTENSION IF EXISTS "uuid-ossp";
