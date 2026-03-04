-- Revert timezone default to 'Asia/Tokyo'
ALTER TABLE events ALTER COLUMN timezone SET DEFAULT 'Asia/Tokyo';
