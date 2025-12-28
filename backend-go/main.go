package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"sago/db"
	"sago/gmail"
	"sago/queue"
	"sago/storage"
)

var gmailService *gmail.Service
var gcsClient *storage.GCSClient
var redisQueue *queue.Client

func main() {
	// Initialize database
	if err := db.Connect(); err != nil {
		log.Printf("Warning: Database not connected: %v", err)
	} else {
		log.Println("Database connected successfully")
		defer db.Close()
	}

	// Initialize Redis queue
	var err error
	redisQueue, err = queue.NewClient()
	if err != nil {
		log.Printf("Warning: Redis not connected: %v", err)
	} else {
		log.Println("Redis queue connected")
		defer redisQueue.Close()
	}

	// Initialize GCS client
	gcsClient, err = storage.NewGCSClient()
	if err != nil {
		log.Printf("Warning: GCS not connected: %v", err)
	} else {
		log.Println("GCS client initialized")
		defer gcsClient.Close()
	}

	// Initialize Gmail service
	initGmailService()

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes
	e.GET("/", hello)
	e.GET("/health", health)

	// Legacy route
	e.POST("/verify", verify)

	// Investor routes
	e.POST("/investors", createInvestor)
	e.GET("/investors", listInvestors)
	e.GET("/investors/:id", getInvestor)

	// Deck & Job routes
	e.POST("/decks/upload", uploadDeck)
	e.GET("/jobs/:id", getJob)
	e.GET("/jobs/:id/report", getJobReport)

	// Gmail routes
	e.GET("/gmail/auth", gmailAuth)
	e.GET("/gmail/check", checkPitchDecks)
	e.POST("/gmail/process/:messageId", processEmail)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}

func initGmailService() {
	var err error
	gmailService, err = gmail.NewService("credentials.json", "token.json")
	if err != nil {
		fmt.Printf("Gmail service not initialized: %v\n", err)
		fmt.Println("Visit /gmail/auth to authenticate")
	} else {
		fmt.Println("Gmail service initialized successfully")
	}
}

// Handlers

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Sago Due Diligence Assistant - Database Ready")
}

func health(c echo.Context) error {
	status := map[string]string{"status": "ok"}

	// Check database
	if db.DB != nil {
		if err := db.DB.Ping(); err == nil {
			status["database"] = "connected"
		} else {
			status["database"] = "error: " + err.Error()
		}
	} else {
		status["database"] = "not initialized"
	}

	// Check Redis
	if redisQueue != nil {
		if err := redisQueue.Ping(context.Background()); err == nil {
			status["redis"] = "connected"
		} else {
			status["redis"] = "error: " + err.Error()
		}
	} else {
		status["redis"] = "not initialized"
	}

	// Check GCS
	if gcsClient != nil {
		status["gcs"] = "connected"
	} else {
		status["gcs"] = "not initialized"
	}

	// Check Gmail
	if gmailService != nil {
		if err := gmailService.HealthCheck(); err == nil {
			status["gmail"] = "connected"
		} else {
			status["gmail"] = "error: " + err.Error()
		}
	} else {
		status["gmail"] = "not initialized"
	}

	return c.JSON(http.StatusOK, status)
}

// Investor handlers

type CreateInvestorRequest struct {
	Email            string   `json:"email"`
	Name             string   `json:"name"`
	InvestmentThesis string   `json:"investment_thesis"`
	FocusAreas       []string `json:"focus_areas"`
	DealBreakers     []string `json:"deal_breakers"`
}

func createInvestor(c echo.Context) error {
	req := new(CreateInvestorRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	investor := &db.Investor{
		Email:            req.Email,
		Name:             &req.Name,
		InvestmentThesis: &req.InvestmentThesis,
		FocusAreas:       req.FocusAreas,
		DealBreakers:     req.DealBreakers,
	}

	if err := db.CreateInvestor(investor); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, investor)
}

func listInvestors(c echo.Context) error {
	investors, err := db.ListInvestors()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, investors)
}

func getInvestor(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid investor id"})
	}

	investor, err := db.GetInvestorByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "investor not found"})
	}

	return c.JSON(http.StatusOK, investor)
}

// Deck upload handler

func uploadDeck(c echo.Context) error {
	// Use MultipartReader to stream the file, avoiding buffer issues
	mr, err := c.Request().MultipartReader()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to start multipart reader", "details": err.Error()})
	}

	var localPath string
	var filename string
	var investorID *uuid.UUID

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("NextPart error: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "processing upload", "details": err.Error()})
		}

		if part.FormName() == "investor_id" {
			// Read investor ID
			buf := new(strings.Builder)
			if _, err := io.Copy(buf, part); err != nil {
				continue
			}
			idStr := buf.String()
			if idStr != "" {
				if id, err := uuid.Parse(idStr); err == nil {
					investorID = &id
				}
			}
		} else if part.FormName() == "file" {
			filename = part.FileName()
			uploadsDir := "uploads"
			if err := os.MkdirAll(uploadsDir, 0755); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create uploads dir"})
			}
			localFilename := fmt.Sprintf("%s_%s", uuid.New().String(), filename)
			localPath = filepath.Join(uploadsDir, localFilename)

			dst, err := os.Create(localPath)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create file"})
			}

			// Stream copy
			if _, err := io.Copy(dst, part); err != nil {
				dst.Close()
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save file", "details": err.Error()})
			}
			dst.Close()

			absPath, _ := filepath.Abs(localPath)
			localPath = absPath
			log.Printf("Saved uploaded file to: %s", localPath)
		}
	}

	if localPath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file required (not found in request)"})
	}

	// Upload to GCS (skipped since we are streaming to disk)
	// If GCS is needed, we should implement UploadFileFromPath
	var gcsPath string

	// Create deck record
	deck := &db.PitchDeck{
		InvestorID: investorID,
		Filename:   filename,
		GCSPath:    &gcsPath,
		Source:     "upload",
	}

	if err := db.CreatePitchDeck(deck); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Create analysis job
	job := &db.AnalysisJob{
		DeckID:     &deck.ID,
		InvestorID: investorID,
	}

	if err := db.CreateJob(job); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Queue job for async processing - pass file path instead of binary content
	if redisQueue != nil {
		err = redisQueue.EnqueueJob(context.Background(), job.ID, investorID, "", localPath)
		if err != nil {
			log.Printf("Failed to enqueue job: %v", err)
		} else {
			log.Printf("Job %s queued successfully with file path: %s", job.ID, localPath)
		}
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"deck":   deck,
		"job_id": job.ID,
		"status": "Job queued for processing",
	})
}

// Job handlers

func getJob(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid job id"})
	}

	job, err := db.GetJobByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "job not found"})
	}

	return c.JSON(http.StatusOK, job)
}

func getJobReport(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid job id"})
	}

	job, err := db.GetJobByID(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "job not found"})
	}

	if job.Status != db.JobStatusCompleted {
		return c.JSON(http.StatusAccepted, map[string]string{
			"status":  job.Status,
			"message": "Job not yet completed",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "completed",
		"report": job.FinalReport,
	})
}

// Legacy verify handler
type VerifyRequest struct {
	DeckContent string `json:"deck_content"`
}

func verify(c echo.Context) error {
	req := new(VerifyRequest)
	if err := c.Bind(req); err != nil {
		return err
	}

	cmd := exec.Command("../engine-python/venv/bin/python", "../engine-python/main.py")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":  err.Error(),
			"output": string(output),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "success",
		"report": string(output),
	})
}

// Gmail Handlers

func gmailAuth(c echo.Context) error {
	if gmailService != nil {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "already_authenticated",
			"message": "Gmail is already connected",
		})
	}

	initGmailService()

	if gmailService != nil {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "success",
			"message": "Gmail connected successfully",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "pending",
		"message": "Check terminal for authentication instructions",
	})
}

func checkPitchDecks(c echo.Context) error {
	if gmailService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Gmail not connected. Visit /gmail/auth first",
		})
	}

	decks, err := gmailService.CheckForPitchDecks()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	var results []map[string]interface{}
	for _, deck := range decks {
		pdfNames := make([]string, len(deck.PDFs))
		for i, pdf := range deck.PDFs {
			pdfNames[i] = pdf.Filename
		}
		results = append(results, map[string]interface{}{
			"message_id": deck.MessageID,
			"subject":    deck.Subject,
			"sender":     deck.Sender,
			"pdfs":       pdfNames,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"count":  len(results),
		"decks":  results,
	})
}

func processEmail(c echo.Context) error {
	if gmailService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Gmail not connected",
		})
	}

	messageID := c.Param("messageId")
	if messageID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "messageId required",
		})
	}

	// Get PDF attachments
	attachments, err := gmailService.GetPDFAttachments(messageID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	if len(attachments) == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "No PDF attachments found",
		})
	}

	// Create pitch deck record in database
	gcsPath := "" // Could upload to GCS here
	sourceMetadata := fmt.Sprintf(`{"message_id": "%s"}`, messageID)
	deck := &db.PitchDeck{
		Filename:       attachments[0].Filename,
		GCSPath:        &gcsPath,
		Source:         "gmail",
		SourceMetadata: &sourceMetadata,
	}

	if err := db.CreatePitchDeck(deck); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create deck: " + err.Error(),
		})
	}

	// Create analysis job
	job := &db.AnalysisJob{
		DeckID: &deck.ID,
	}

	if err := db.CreateJob(job); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create job: " + err.Error(),
		})
	}

	// Save Gmail attachment locally for Python worker
	uploadsDir := "uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Printf("Failed to create uploads dir: %v", err)
	}
	localFilename := fmt.Sprintf("%s_%s", uuid.New().String(), attachments[0].Filename)
	localPath := filepath.Join(uploadsDir, localFilename)
	if err := os.WriteFile(localPath, attachments[0].Data, 0644); err != nil {
		log.Printf("Failed to save gmail attachment locally: %v", err)
		localPath = ""
	} else {
		absPath, _ := filepath.Abs(localPath)
		localPath = absPath
		log.Printf("Saved gmail attachment to: %s", localPath)
	}

	// Queue job to Redis with the PDF file path
	if redisQueue != nil {
		err = redisQueue.EnqueueJob(context.Background(), job.ID, nil, "", localPath)
		if err != nil {
			log.Printf("Failed to enqueue job: %v", err)
		} else {
			log.Printf("Gmail job %s queued successfully with path: %s", job.ID, localPath)
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "queued",
		"job_id":   job.ID,
		"deck_id":  deck.ID,
		"pdf_name": attachments[0].Filename,
	})
}
