# System Architecture: AI-Powered Due Diligence Assistant

## 1. Overview
The AI-Powered Due Diligence Assistant is a full-stack application designed to automate the initial screening of startup pitch decks. It combines a robust Go backend for reliable file handling with a sophisticated Python AI engine for deep analysis.

### High-Level Architecture
![System Architecture](docs/images/architecture_diagram.png)

```mermaid
graph TD
    User[User / Investor] -->|Uploads PDF| FE[Frontend (React)]
    FE -->|POST /decks/upload| BE[Backend API (Go)]
    BE -->|Streams File| FS[Local Filesystem / Uploads]
    BE -->|Enqueues Job| Redis[(Redis Queue)]
    
    subgraph "AI Engine (Python)"
        Worker[Worker Process] -->|1. Dequeues Job| Redis
        Worker -->|2. Reads PDF| FS
        Worker -->|3. OCR & Extraction| OCR[PyTesseract/PyPDF]
        Worker -->|4. Sync & Context| Pinecone[(Vector DB)]
        Worker -->|5. Agent Analysis| CrewAI[CrewAI Agents]
        Worker -->|6. Saves Report| Postgres[(PostgreSQL DB)]
    end
    
    BE -->|Reads Status/Report| Postgres
    FE -->|Polls Status| BE
```

## 2. Technology Stack

### Frontend
- **Framework:** React (Vite) + TypeScript
- **Styling:** CSS Modules / Vanilla CSS (Modern Dark Mode Design)
- **Key Libraries:** `react-markdown` (Report Rendering), `lucide-react` (Icons)
- **Responsibility:** User interface for uploading decks, managing investor profiles, and viewing analysis reports.

### Backend ("Sago")
- **Language:** Go (Golang) 1.22+
- **Framework:** Echo v4
- **Key Features:**
    - **Streaming Multipart Uploads:** Uses `MultipartReader` to handle large binary files (e.g., image-heavy PDFs) robustly without memory exhaustion.
    - **Job Queueing:** Pushes analysis tasks to Redis list `sago:jobs`.
    - **Data Access:** Direct connection to PostgreSQL for CRUD on Decks, Investors, and Jobs.

### AI Engine ("Engine")
- **Language:** Python 3.10+
- **Orchestrator:** CrewAI (Multi-Agent System)
- **Agents:**
    1.  **Scribe:** Extracts textual content from PDFs (supports OCR fallback for image-only PDFs like `shopify-pitch-deck.pdf`).
    2.  **Researcher:** Verifies claims using search tools (SerperDev) and gathers market data.
    3.  **Analyst:** Synthesizes findings into a structured investment memo, personalized based on Investor Thesis.
- **Memory/Context:**
    - **Pinecone (Vector DB):** Stores and retrieves investor profiles and historical contexts (`sago-investors` index). Syncs SQL profile data to embeddings (`bge-large-en-v1.5`) for semantic retrieval.
- **Dependencies:** `pypdf`, `pdf2image`, `pytesseract`, `pinecone-client`, `sentence-transformers`, `python-dotenv`.

### Infrastructure
- **Database:** PostgreSQL (Primary relational store)
- **Queue:** Redis (Job orchestration)
- **Storage:** Local Filesystem (`uploads/`), extensible to GCS.

---

## 3. Core Workflows

### A. Pitch Deck Upload & Processing
1.  **Upload:** User selects a PDF. Frontend sends it via `multipart/form-data`.
2.  **Streaming:** Go backend uses a streaming iterator to save the file to disk chunk-by-chunk, avoiding EOF errors common with large files in standard parsers.
3.  **Queueing:** A job payload `{ "job_id": "...", "deck_path": "..." }` is pushed to Redis.
4.  **Extraction:** Python worker picks up the job. It attempts text extraction via `pypdf`. If extracted text is insufficient (<100 chars), it falls back to **OCR** (Tesseract) to read text from page images.
5.  **Mock Fallback:** *Removed in production.* The system fails gracefully with an explicit error if no text can be read, ensuring no hallucinated "mock" data appears.

### B. Agentic Analysis
1.  **Context Retrieval:** 
    - Worker checks `investor_id`.
    - Fetches profile from SQL.
    - **Syncs** profile to Pinecone.
    - Retrieves semantic context (Thesis, Deal Breakers) from Pinecone to guide agents.
2.  **Crew Execution:**
    - **Researcher** validates claims extracted by Scribe (e.g., "Revenue $2M").
    - **Analyst** writes the report, citing sources and highlighting discrepancies.
3.  **Completion:** Final report is saved to Postgres `analysis_jobs` table.

## 4. Database Schema
(Simplified)

**`investors`**
- `id` (UUID)
- `name`, `email`
- `investment_thesis` (Text)
- `focus_areas`, `deal_breakers` (Arrays)

**`pitch_decks`**
- `id` (UUID)
- `filename`
- `source` ("upload" / "email")

**`analysis_jobs`**
- `id` (UUID)
- `status` ("queued", "running", "completed", "failed")
- `final_report` (Markdown Text)

## 5. Recent Improvements
- **Robust Uploads:** Switched from `ParseMultipartForm` to manual `MultipartReader` stream processing to fix `unexpected EOF` errors on large files.
- **Vector Integration:** Added `InvestorMemory` class to bridge SQL investor data with Pinecone for semantic personalization.
- **OCR Support:** Integrated `pdf2image` + `pytesseract` to handle image-based decks correctly.
