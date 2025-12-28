-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Investors table
CREATE TABLE IF NOT EXISTS investors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    investment_thesis TEXT,
    focus_areas TEXT[],
    deal_breakers TEXT[],
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Pitch Decks table
CREATE TABLE IF NOT EXISTS pitch_decks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investor_id UUID REFERENCES investors(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    gcs_path TEXT,
    file_hash VARCHAR(64),
    source VARCHAR(50) DEFAULT 'upload',
    source_metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Analysis Jobs table
CREATE TABLE IF NOT EXISTS analysis_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deck_id UUID REFERENCES pitch_decks(id) ON DELETE CASCADE,
    investor_id UUID REFERENCES investors(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'pending',
    claims_extracted JSONB,
    verification_results JSONB,
    final_report TEXT,
    final_report_gcs_path TEXT,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_jobs_status ON analysis_jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_investor ON analysis_jobs(investor_id);
CREATE INDEX IF NOT EXISTS idx_decks_investor ON pitch_decks(investor_id);

-- Insert demo investor
INSERT INTO investors (email, name, investment_thesis, focus_areas, deal_breakers)
VALUES (
    'demo@sago.ai',
    'Demo Investor',
    'Focus on B2B SaaS companies with strong recurring revenue',
    ARRAY['B2B SaaS', 'Enterprise Software', 'Developer Tools'],
    ARRAY['Burn rate > 3x revenue', 'No competitive moat', 'TAM under $1B']
) ON CONFLICT (email) DO NOTHING;
