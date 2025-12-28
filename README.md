# 🔍 Sago - AI-Powered Due Diligence Assistant

An AI-powered platform that automates pitch deck analysis for venture capital due diligence. Upload a pitch deck and get a comprehensive report with extracted claims, fact-checked verification, and critical analysis—all powered by multi-agent AI.

![Sago Architecture](docs/images/architecture_diagram.png)

## ✨ Features

- **📄 Pitch Deck Analysis** - Upload PDF or text files for automated analysis
- **🤖 Multi-Agent AI Pipeline** - Three specialized agents work together:
  - **Scribe** - Extracts specific, verifiable claims
  - **Researcher** - Fact-checks claims via web search
  - **Analyst** - Generates critical due diligence report
- **📧 Gmail Integration** - Scan inbox for pitch deck emails and analyze directly
- **👤 Investor Profiles** - Personalized analysis based on investment thesis
- **⚡ Async Processing** - Redis-powered job queue for non-blocking analysis
- **🎨 Modern UI** - React + TypeScript frontend with dark theme

## 📦 Sample Input & Output

### Sample Input
Use the included sample pitch deck to test the system:
- **`mock_pitch_deck.txt`** - Shopify pitch deck (text format, included in repo)
- **`shopify-pitch-deck.pdf`** - Shopify pitch deck (PDF format, included in repo)

### Sample Output
See `docs/sample_output.md` for a complete analysis report. Key sections include:

**Extracted Claims:**
```
• Active Shopify Merchants: 200,000+
• Gross Merchandise Volume (GMV) in Q3 '15: $1.9B+
• Global Total Addressable Market (TAM): $46B
```

**Verification Results:**
```
• 200,000+ merchants → CONTRADICTED (now 2M+ as of 2023)
• $1.9B GMV Q3'15 → CONFIRMED (official financial reports)
• $46B TAM → UNVERIFIED (requires methodology review)
```

**Red Flags Identified:**
```
• Vague metrics ("200,000+" lacks precision)
• Vanity metrics (GMV vs. actual revenue)
• Celebrity endorsements as social proof (misleading)
• Aggressive TAM estimates
```


## 🏗️ Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  React Frontend │────▶│   Go Backend    │────▶│  Python Worker  │
│  localhost:5173 │     │  localhost:8080 │     │    (CrewAI)     │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │                        │
                    ┌──────────┴──────────┐            │
                    │                     │            │
              ┌─────▼─────┐        ┌──────▼─────┐      │
              │ PostgreSQL │        │   Redis    │◀────┘
              │   :5433    │        │   :6380    │
              └────────────┘        └────────────┘
```

## 📋 Prerequisites

- **Go** 1.21+
- **Python** 3.13+
- **Node.js** 22+
- **Docker** & Docker Compose
- **API Keys** (see below)

## 🔑 Required API Keys

| Service | Purpose | Get Key |
|---------|---------|---------|
| **OpenAI** | LLM for analysis | [platform.openai.com](https://platform.openai.com/api-keys) |
| **Serper** | Web search for verification | [serper.dev](https://serper.dev) |
| **Pinecone** (optional) | Vector DB for investor memory | [pinecone.io](https://www.pinecone.io) |
| **Gmail OAuth** (optional) | Email integration | [Google Cloud Console](https://console.cloud.google.com) |

## 🚀 Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/YOUR_USERNAME/AI-Powered-Due-Diligence-Assistant.git
cd AI-Powered-Due-Diligence-Assistant
```

### 2. Start Infrastructure (PostgreSQL + Redis)

```bash
docker compose up -d
```

This starts:
- PostgreSQL on `localhost:5433`
- Redis on `localhost:6380`

### 3. Initialize the Database

```bash
# Run the migration script
docker exec -i sago-postgres psql -U sago -d sago < backend-go/db/migrations/001_init.sql
```

### 4. Configure Environment Variables

```bash
# Copy the example env file
cp engine-python/.env.example engine-python/.env

# Edit with your API keys
nano engine-python/.env
```

**Required variables:**
```env
# OpenAI API Key (required)
OPENAI_API_KEY=sk-...

# Serper API Key (required for fact-checking)
SERPER_API_KEY=your-serper-key

# Database (default works with docker-compose)
DATABASE_URL=postgresql://sago:sago_dev_password@localhost:5433/sago

# Redis (default works with docker-compose)
REDIS_URL=redis://localhost:6380
```

### 5. Start the Go Backend

```bash
cd backend-go
go mod download
go build -o sago && ./sago
```

You should see:
```
Database connected successfully
Redis queue connected
⇨ http server started on [::]:8080
```

### 6. Start the Python Worker

```bash
cd engine-python
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
python worker.py
```

You should see:
```
[Worker] Starting job worker, listening on sago:jobs...
[Worker] Redis: redis://localhost:6380
```

### 7. Start the Frontend

```bash
cd frontend
npm install
npm run dev
```

Open **http://localhost:5173** in your browser! 🎉

## 📖 Usage

### Upload a Pitch Deck

1. Go to http://localhost:5173
2. Click **Analysis** tab
3. Drag & drop a PDF or TXT file
4. Watch the analysis progress in real-time

### Create an Investor Profile (Optional)

1. Click **Investors** tab
2. Click **+ New**
3. Fill in:
   - Email & Name
   - Investment Thesis
   - Focus Areas (tags)
   - Deal Breakers (red flags)
4. Select the investor before uploading for personalized analysis

### Gmail Integration (Optional)

1. Set up Gmail OAuth (see below)
2. Click **Gmail** tab
3. Click **🔄 Refresh** to scan inbox
4. Click **Analyze** on any pitch deck email

## 📧 Gmail OAuth Setup (Optional)

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project or select existing
3. Enable the **Gmail API**
4. Configure OAuth consent screen
5. Create OAuth 2.0 credentials (Desktop app)
6. Download as `credentials.json` and place in `backend-go/`
7. Restart the backend - it will prompt for authentication

## 🗂️ Project Structure

```
AI-Powered-Due-Diligence-Assistant/
├── backend-go/           # Go API server
│   ├── main.go           # Entry point & routes
│   ├── db/               # Database models & migrations
│   ├── queue/            # Redis job queue
│   ├── gmail/            # Gmail integration
│   └── storage/          # GCS client (optional)
├── engine-python/        # Python AI worker
│   ├── worker.py         # Redis job consumer
│   ├── agents/           # CrewAI agents
│   │   ├── scribe.py     # Claim extraction
│   │   ├── researcher.py # Web verification
│   │   └── analyst.py    # Due diligence report
│   └── personalization/  # Investor memory (Pinecone)
├── frontend/             # React + TypeScript UI
│   ├── src/
│   │   ├── App.tsx       # Main component
│   │   ├── api.ts        # Backend API client
│   │   └── index.css     # Styling
│   └── vite.config.ts    # Dev server config
└── docker-compose.yml    # PostgreSQL + Redis
```

## 🛠️ API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Service health check |
| POST | `/decks/upload` | Upload pitch deck for analysis |
| GET | `/jobs/:id` | Get job status and report |
| GET | `/investors` | List investor profiles |
| POST | `/investors` | Create investor profile |
| GET | `/gmail/check` | Scan Gmail for pitch decks |
| POST | `/gmail/process/:id` | Analyze Gmail pitch deck |

## 🔧 Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | ✅ | - | OpenAI API key |
| `SERPER_API_KEY` | ✅ | - | Serper search API key |
| `DATABASE_URL` | ✅ | - | PostgreSQL connection string |
| `REDIS_URL` | ✅ | - | Redis connection string |
| `PINECONE_API_KEY` | ❌ | - | Pinecone for investor memory |
| `PINECONE_INDEX` | ❌ | - | Pinecone index name |
| `GCS_BUCKET` | ❌ | - | GCS bucket for file storage |

### Using Different LLMs

The system supports multiple LLM providers. Edit `engine-python/agents/scribe.py`:

```python
# OpenAI (default)
LLM(model="openai/gpt-4o-mini", api_key=os.getenv("OPENAI_API_KEY"))

# Google Gemini
LLM(model="gemini/gemini-2.0-flash", api_key=os.getenv("GOOGLE_API_KEY"))

# Local Ollama
LLM(model="ollama/llama3.2", base_url="http://localhost:11434")
```

## 🐳 Docker Deployment

Build and run all services with Docker Compose:

```bash
# Build images
docker compose -f docker-compose.prod.yml build

# Start all services
docker compose -f docker-compose.prod.yml up -d
```

## 📊 Cost Estimates

| Service | Cost |
|---------|------|
| OpenAI GPT-4o-mini | ~$0.01 per analysis |
| Serper API | Free tier: 2,500 searches/month |
| Pinecone | Free tier: 100K vectors |
| PostgreSQL | Self-hosted (free) |
| Redis | Self-hosted (free) |

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing`)
5. Open a Pull Request

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- [CrewAI](https://github.com/joaomdmoura/crewai) - Multi-agent framework
- [Echo](https://echo.labstack.com/) - Go web framework
- [Vite](https://vitejs.dev/) - Frontend build tool