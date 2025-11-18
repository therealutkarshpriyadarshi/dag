-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- DAGs table
CREATE TABLE dags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    schedule VARCHAR(100), -- Cron expression
    is_paused BOOLEAN DEFAULT false,
    tags JSONB DEFAULT '[]'::jsonb,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on dag name for faster lookups
CREATE INDEX idx_dags_name ON dags(name);
CREATE INDEX idx_dags_is_paused ON dags(is_paused);
CREATE INDEX idx_dags_tags ON dags USING GIN(tags);

-- DAG runs table
CREATE TABLE dag_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    dag_id UUID NOT NULL REFERENCES dags(id) ON DELETE CASCADE,
    execution_date TIMESTAMP NOT NULL,
    state VARCHAR(50) NOT NULL DEFAULT 'queued',
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    external_trigger BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version INTEGER NOT NULL DEFAULT 1, -- For optimistic locking
    CONSTRAINT unique_dag_execution UNIQUE (dag_id, execution_date)
);

-- Create indexes for dag_runs
CREATE INDEX idx_dag_runs_dag_id ON dag_runs(dag_id);
CREATE INDEX idx_dag_runs_state ON dag_runs(state);
CREATE INDEX idx_dag_runs_execution_date ON dag_runs(execution_date);
CREATE INDEX idx_dag_runs_created_at ON dag_runs(created_at DESC);

-- Task instances table
CREATE TABLE task_instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id VARCHAR(255) NOT NULL,
    dag_run_id UUID NOT NULL REFERENCES dag_runs(id) ON DELETE CASCADE,
    state VARCHAR(50) NOT NULL DEFAULT 'queued',
    try_number INTEGER NOT NULL DEFAULT 1,
    max_tries INTEGER NOT NULL DEFAULT 1,
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    duration INTERVAL,
    hostname VARCHAR(255),
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version INTEGER NOT NULL DEFAULT 1, -- For optimistic locking
    CONSTRAINT unique_task_per_run UNIQUE (task_id, dag_run_id, try_number)
);

-- Create indexes for task_instances
CREATE INDEX idx_task_instances_dag_run_id ON task_instances(dag_run_id);
CREATE INDEX idx_task_instances_task_id ON task_instances(task_id);
CREATE INDEX idx_task_instances_state ON task_instances(state);
CREATE INDEX idx_task_instances_created_at ON task_instances(created_at DESC);

-- Task logs table
CREATE TABLE task_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_instance_id UUID NOT NULL REFERENCES task_instances(id) ON DELETE CASCADE,
    log_data TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index for task_logs
CREATE INDEX idx_task_logs_task_instance_id ON task_logs(task_instance_id);
CREATE INDEX idx_task_logs_timestamp ON task_logs(timestamp DESC);

-- State history table for audit trail
CREATE TABLE state_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type VARCHAR(50) NOT NULL, -- 'dag_run' or 'task_instance'
    entity_id UUID NOT NULL,
    old_state VARCHAR(50),
    new_state VARCHAR(50) NOT NULL,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Create indexes for state_history
CREATE INDEX idx_state_history_entity ON state_history(entity_type, entity_id);
CREATE INDEX idx_state_history_changed_at ON state_history(changed_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to auto-update updated_at
CREATE TRIGGER update_dags_updated_at BEFORE UPDATE ON dags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_dag_runs_updated_at BEFORE UPDATE ON dag_runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_task_instances_updated_at BEFORE UPDATE ON task_instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
