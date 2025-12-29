-- Create participants table
CREATE TABLE IF NOT EXISTS participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    employee_id VARCHAR(255),
    phone VARCHAR(50),
    qr_email VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'tentative',
    qr_code VARCHAR(255) UNIQUE NOT NULL,
    qr_code_generated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB,
    payment_status VARCHAR(50) DEFAULT 'unpaid',
    payment_amount NUMERIC(10, 2),
    payment_date TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_event_email UNIQUE(event_id, email)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_participants_event_id ON participants(event_id);
CREATE INDEX IF NOT EXISTS idx_participants_employee_id ON participants(employee_id);
CREATE INDEX IF NOT EXISTS idx_participants_email ON participants(email);
CREATE INDEX IF NOT EXISTS idx_participants_qr_email ON participants(qr_email) WHERE qr_email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_participants_qr_code ON participants(qr_code);
CREATE INDEX IF NOT EXISTS idx_participants_status ON participants(status);
CREATE INDEX IF NOT EXISTS idx_participants_payment_status ON participants(payment_status);
CREATE INDEX IF NOT EXISTS idx_participants_created_at ON participants(created_at);
CREATE INDEX IF NOT EXISTS idx_participants_metadata ON participants USING gin(metadata);
