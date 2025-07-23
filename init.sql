-- Database initialization script for Claude Control System

-- Create tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hook_type VARCHAR(50) NOT NULL,
    task_data JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    action_taken VARCHAR(50),
    response_data JSONB
);

-- Create task history table
CREATE TABLE IF NOT EXISTS task_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_hook_type ON tasks(hook_type);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);
CREATE INDEX IF NOT EXISTS idx_task_history_task_id ON task_history(task_id);
CREATE INDEX IF NOT EXISTS idx_task_history_created_at ON task_history(created_at);

-- Insert some sample data for testing (optional)
-- INSERT INTO tasks (hook_type, task_data, status) VALUES 
-- ('PreToolUse', '{"tool": "Bash", "command": "ls -la"}', 'pending'),
-- ('Notification', '{"message": "Claude needs attention"}', 'pending');

-- Grant necessary permissions
GRANT ALL PRIVILEGES ON DATABASE claude_control TO claude_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO claude_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO claude_user;