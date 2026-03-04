-- Change timezone default from 'Asia/Tokyo' to 'UTC'
ALTER TABLE events ALTER COLUMN timezone SET DEFAULT 'UTC';
