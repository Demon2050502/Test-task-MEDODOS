CREATE TABLE IF NOT EXISTS tasks (
	id BIGSERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	priority TEXT NOT NULL DEFAULT 'normal',
	scheduled_for DATE NULL,
	source_recurring_task_id BIGINT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);
CREATE INDEX IF NOT EXISTS idx_tasks_scheduled_for ON tasks (scheduled_for DESC);
CREATE INDEX IF NOT EXISTS idx_tasks_source_recurring_task_id ON tasks (source_recurring_task_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_recurring_unique_schedule
	ON tasks (source_recurring_task_id, scheduled_for)
	WHERE source_recurring_task_id IS NOT NULL AND scheduled_for IS NOT NULL;

CREATE TABLE IF NOT EXISTS recurring_tasks (
	id BIGSERIAL PRIMARY KEY,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	priority TEXT NOT NULL DEFAULT 'normal',
	start_date DATE NOT NULL,
	rule_type TEXT NOT NULL,
	every_n_days INTEGER NULL,
	day_of_month INTEGER NULL,
	parity TEXT NULL,
	last_generated_for DATE NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recurring_tasks_start_date ON recurring_tasks (start_date);

CREATE TABLE IF NOT EXISTS recurring_task_dates (
	recurring_task_id BIGINT NOT NULL REFERENCES recurring_tasks(id) ON DELETE CASCADE,
	scheduled_date DATE NOT NULL,
	PRIMARY KEY (recurring_task_id, scheduled_date)
);
