-- Drop indexes
DROP INDEX IF EXISTS idx_checkins_checked_in_by;
DROP INDEX IF EXISTS idx_checkins_checked_in_at;
DROP INDEX IF EXISTS idx_checkins_participant_id;
DROP INDEX IF EXISTS idx_checkins_event_id;

-- Drop checkins table
DROP TABLE IF EXISTS checkins;
