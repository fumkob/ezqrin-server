-- Create checkins table
CREATE TABLE IF NOT EXISTS checkins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    participant_id UUID NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    checked_in_at TIMESTAMP NOT NULL DEFAULT NOW(),
    checked_in_by UUID REFERENCES users(id),
    checkin_method VARCHAR(50) NOT NULL DEFAULT 'qrcode',
    device_info JSONB,

    CONSTRAINT unique_event_participant_checkin UNIQUE(event_id, participant_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_checkins_event_id ON checkins(event_id);
CREATE INDEX IF NOT EXISTS idx_checkins_participant_id ON checkins(participant_id);
CREATE INDEX IF NOT EXISTS idx_checkins_checked_in_at ON checkins(checked_in_at);
CREATE INDEX IF NOT EXISTS idx_checkins_checked_in_by ON checkins(checked_in_by);
