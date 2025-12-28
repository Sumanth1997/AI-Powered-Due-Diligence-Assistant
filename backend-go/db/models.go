package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Investor represents an investor profile
type Investor struct {
	ID               uuid.UUID      `db:"id" json:"id"`
	Email            string         `db:"email" json:"email"`
	Name             *string        `db:"name" json:"name,omitempty"`
	InvestmentThesis *string        `db:"investment_thesis" json:"investment_thesis,omitempty"`
	FocusAreas       pq.StringArray `db:"focus_areas" json:"focus_areas,omitempty"`
	DealBreakers     pq.StringArray `db:"deal_breakers" json:"deal_breakers,omitempty"`
	Notes            *string        `db:"notes" json:"notes,omitempty"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updated_at"`
}

// PitchDeck represents an uploaded pitch deck
type PitchDeck struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	InvestorID     *uuid.UUID `db:"investor_id" json:"investor_id,omitempty"`
	Filename       string     `db:"filename" json:"filename"`
	GCSPath        *string    `db:"gcs_path" json:"gcs_path,omitempty"`
	FileHash       *string    `db:"file_hash" json:"file_hash,omitempty"`
	Source         string     `db:"source" json:"source"`
	SourceMetadata *string    `db:"source_metadata" json:"source_metadata,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

// AnalysisJob represents a due diligence analysis job
type AnalysisJob struct {
	ID                  uuid.UUID  `db:"id" json:"id"`
	DeckID              *uuid.UUID `db:"deck_id" json:"deck_id,omitempty"`
	InvestorID          *uuid.UUID `db:"investor_id" json:"investor_id,omitempty"`
	Status              string     `db:"status" json:"status"`
	ClaimsExtracted     *string    `db:"claims_extracted" json:"claims_extracted,omitempty"`
	VerificationResults *string    `db:"verification_results" json:"verification_results,omitempty"`
	FinalReport         *string    `db:"final_report" json:"final_report,omitempty"`
	FinalReportGCSPath  *string    `db:"final_report_gcs_path" json:"final_report_gcs_path,omitempty"`
	ErrorMessage        *string    `db:"error_message" json:"error_message,omitempty"`
	StartedAt           *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt         *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
}

// Job status constants
const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
)
