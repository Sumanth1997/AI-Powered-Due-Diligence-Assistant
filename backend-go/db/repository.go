package db

import (
	"github.com/google/uuid"
)

// CreateInvestor creates a new investor and returns the ID
func CreateInvestor(investor *Investor) error {
	query := `
		INSERT INTO investors (email, name, investment_thesis, focus_areas, deal_breakers, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	return DB.QueryRowx(query,
		investor.Email,
		investor.Name,
		investor.InvestmentThesis,
		investor.FocusAreas,
		investor.DealBreakers,
		investor.Notes,
	).Scan(&investor.ID, &investor.CreatedAt, &investor.UpdatedAt)
}

// GetInvestorByID retrieves an investor by ID
func GetInvestorByID(id uuid.UUID) (*Investor, error) {
	var investor Investor
	err := DB.Get(&investor, "SELECT * FROM investors WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &investor, nil
}

// GetInvestorByEmail retrieves an investor by email
func GetInvestorByEmail(email string) (*Investor, error) {
	var investor Investor
	err := DB.Get(&investor, "SELECT * FROM investors WHERE email = $1", email)
	if err != nil {
		return nil, err
	}
	return &investor, nil
}

// ListInvestors returns all investors
func ListInvestors() ([]Investor, error) {
	var investors []Investor
	err := DB.Select(&investors, "SELECT * FROM investors ORDER BY created_at DESC")
	return investors, err
}

// CreatePitchDeck creates a new pitch deck record
func CreatePitchDeck(deck *PitchDeck) error {
	query := `
		INSERT INTO pitch_decks (investor_id, filename, gcs_path, file_hash, source, source_metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	return DB.QueryRowx(query,
		deck.InvestorID,
		deck.Filename,
		deck.GCSPath,
		deck.FileHash,
		deck.Source,
		deck.SourceMetadata,
	).Scan(&deck.ID, &deck.CreatedAt)
}

// GetPitchDeckByID retrieves a pitch deck by ID
func GetPitchDeckByID(id uuid.UUID) (*PitchDeck, error) {
	var deck PitchDeck
	err := DB.Get(&deck, "SELECT * FROM pitch_decks WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &deck, nil
}

// CreateJob creates a new analysis job
func CreateJob(job *AnalysisJob) error {
	query := `
		INSERT INTO analysis_jobs (deck_id, investor_id, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	return DB.QueryRowx(query,
		job.DeckID,
		job.InvestorID,
		JobStatusPending,
	).Scan(&job.ID, &job.CreatedAt)
}

// GetJobByID retrieves a job by ID
func GetJobByID(id uuid.UUID) (*AnalysisJob, error) {
	var job AnalysisJob
	err := DB.Get(&job, "SELECT * FROM analysis_jobs WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateJobStatus updates the status of a job
func UpdateJobStatus(id uuid.UUID, status string) error {
	query := `UPDATE analysis_jobs SET status = $1 WHERE id = $2`
	_, err := DB.Exec(query, status, id)
	return err
}

// UpdateJobStarted marks a job as started
func UpdateJobStarted(id uuid.UUID) error {
	query := `UPDATE analysis_jobs SET status = $1, started_at = NOW() WHERE id = $2`
	_, err := DB.Exec(query, JobStatusRunning, id)
	return err
}

// UpdateJobCompleted marks a job as completed with results
func UpdateJobCompleted(id uuid.UUID, claims, verification, finalReport string) error {
	query := `
		UPDATE analysis_jobs 
		SET status = $1, claims_extracted = $2, verification_results = $3, 
		    final_report = $4, completed_at = NOW()
		WHERE id = $5`
	_, err := DB.Exec(query, JobStatusCompleted, claims, verification, finalReport, id)
	return err
}

// UpdateJobFailed marks a job as failed with error message
func UpdateJobFailed(id uuid.UUID, errorMsg string) error {
	query := `
		UPDATE analysis_jobs 
		SET status = $1, error_message = $2, completed_at = NOW()
		WHERE id = $3`
	_, err := DB.Exec(query, JobStatusFailed, errorMsg, id)
	return err
}

// ListJobsByInvestor lists all jobs for an investor
func ListJobsByInvestor(investorID uuid.UUID) ([]AnalysisJob, error) {
	var jobs []AnalysisJob
	err := DB.Select(&jobs, "SELECT * FROM analysis_jobs WHERE investor_id = $1 ORDER BY created_at DESC", investorID)
	return jobs, err
}
