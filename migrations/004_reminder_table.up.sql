CREATE TABLE IF NOT EXISTS reminders (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	trip_id BIGINT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
	user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	message VARCHAR(255), 
	trigger_at TIMESTAMP WITH TIME ZONE NOT NULL,
	status VARCHAR(255) NOT NULL DEFAULT 'pending',
	CONSTRAINT check_status CHECK (status IN ('pending', 'sent', 'failed', 'cancelled'))
)

CREATE INDEX IF NOT EXISTS idx_reminders_trigger_at_pending 
ON reminders(trigger_at) 
WHERE status = 'pending';