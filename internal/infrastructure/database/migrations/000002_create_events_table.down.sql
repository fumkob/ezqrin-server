-- Drop indexes
DROP INDEX IF EXISTS idx_events_created_at;
DROP INDEX IF EXISTS idx_events_status;
DROP INDEX IF EXISTS idx_events_start_date;
DROP INDEX IF EXISTS idx_events_organizer_id;

-- Drop events table
DROP TABLE IF EXISTS events;
