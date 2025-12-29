-- Drop indexes
DROP INDEX IF EXISTS idx_participants_metadata;
DROP INDEX IF EXISTS idx_participants_created_at;
DROP INDEX IF EXISTS idx_participants_payment_status;
DROP INDEX IF EXISTS idx_participants_status;
DROP INDEX IF EXISTS idx_participants_qr_code;
DROP INDEX IF EXISTS idx_participants_qr_email;
DROP INDEX IF EXISTS idx_participants_email;
DROP INDEX IF EXISTS idx_participants_employee_id;
DROP INDEX IF EXISTS idx_participants_event_id;

-- Drop participants table
DROP TABLE IF EXISTS participants;
