"""
Redis Job Worker
Listens for analysis jobs and processes them with CrewAI agents.
"""
import os
import sys
import json
import redis
import time
from dotenv import load_dotenv

load_dotenv()

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from db.models import SessionLocal, get_job_by_id, get_investor_by_id
from db.models import update_job_started, update_job_completed, update_job_failed

# Redis connection
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6380")
redis_client = redis.from_url(REDIS_URL)

# Queue name
JOB_QUEUE = "sago:jobs"


def run_analysis(job_id: str, investor_id: str = None, deck_content: str = None, deck_path: str = None):
    """Run the CrewAI analysis pipeline."""
    from crewai import Task, Crew, Process, LLM
    from agents.scribe import create_scribe_agent
    from agents.researcher import create_researcher_agent
    from agents.analyst import create_analyst_agent
    
    print(f"[Worker] Starting analysis for job {job_id}")
    
    # Configure Gemini LLM
    gemini_api_key = os.getenv("GOOGLE_API_KEY")
    if gemini_api_key and gemini_api_key != 'YOUR_GEMINI_API_KEY_HERE':
        print("[Worker] Using Google Gemini API")
        # CrewAI will use this via environment variable
        os.environ["OPENAI_API_KEY"] = "NA"  # Disable OpenAI
    else:
        print("[Worker] Warning: GOOGLE_API_KEY not set, falling back to local LLM")
    
    db = SessionLocal()
    
    try:
        # Mark job as started
        update_job_started(db, job_id)
        
        # Get investor context for personalization
        investor_context = None

        if investor_id:
            investor = get_investor_by_id(db, investor_id)
            if investor:
                # 1. Start with SQL data
                parts = []
                if investor.focus_areas:
                    parts.append(f"Focus Areas: {', '.join(investor.focus_areas)}")
                if investor.deal_breakers:
                    parts.append(f"Deal Breakers: {', '.join(investor.deal_breakers)}")
                if investor.investment_thesis:
                    parts.append(f"Investment Thesis: {investor.investment_thesis}")
                
                sql_context = "\n".join(parts)
                print(f"[Worker] Loaded SQL context for investor {investor_id}")
                
                # 2. Try Vector DB (Memory)
                try:
                    from personalization.investor_memory import InvestorMemory
                    memory = InvestorMemory()
                    
                    # Sync Profile to Vector DB (Lazy Sync)
                    profile = {
                        "thesis": investor.investment_thesis or "",
                        "deal_breakers": investor.deal_breakers or [],
                        "focus_areas": investor.focus_areas or [],
                        "notes": investor.notes or ""
                    }
                    memory.store_investor_profile(investor_id, profile)
                    
                    # Get Focus from Memory (retrieves formatted string)
                    vector_context = memory.get_investor_focus(investor_id)
                    
                    if vector_context:
                        investor_context = vector_context
                        print(f"[Worker] Using Vector DB context for personalization")
                    else:
                        investor_context = sql_context
                        
                except Exception as e:
                    print(f"[Worker] Vector DB warning: {e}. Falling back to SQL.")
                    investor_context = sql_context
        
        # Debug: Log what we received
        print(f"[Worker] DEBUG: deck_path = {deck_path}")
        print(f"[Worker] DEBUG: deck_path exists = {os.path.exists(deck_path) if deck_path else 'N/A'}")
        print(f"[Worker] DEBUG: deck_content length = {len(deck_content) if deck_content else 0}")
        
        # Priority 1: Read from deck_path if provided (direct file path)
        if deck_path and os.path.exists(deck_path):
            print(f"[Worker] Reading PDF from path: {deck_path}")
            try:
                from pypdf import PdfReader
                reader = PdfReader(deck_path)
                text_parts = []
                for page in reader.pages:
                    text = page.extract_text()
                    if text:
                        text_parts.append(text)
                deck_content = "\n\n".join(text_parts)
                print(f"[Worker] Extracted {len(deck_content)} chars from {len(reader.pages)} pages")
                
                # If no text extracted, try OCR
                if len(deck_content.strip()) < 100:
                    print("[Worker] No text found, attempting OCR...")
                    try:
                        from pdf2image import convert_from_path
                        import pytesseract
                        
                        images = convert_from_path(deck_path, dpi=150)
                        ocr_text_parts = []
                        for i, img in enumerate(images):
                            text = pytesseract.image_to_string(img)
                            if text.strip():
                                ocr_text_parts.append(text)
                            print(f"[Worker] OCR page {i+1}: {len(text)} chars")
                        
                        if ocr_text_parts:
                            deck_content = "\n\n".join(ocr_text_parts)
                            print(f"[Worker] OCR extracted {len(deck_content)} chars from {len(images)} pages")
                    except Exception as ocr_error:
                        print(f"[Worker] OCR failed: {ocr_error}")
                        
            except Exception as e:
                print(f"[Worker] PDF extraction from path failed: {e}")
                # Try reading as text file
                try:
                    with open(deck_path, 'r', encoding='utf-8') as f:
                        deck_content = f.read()
                except:
                    deck_content = f"Error reading file: {e}"
        # Priority 2: Use provided deck_content
        elif deck_content and deck_content.strip():
            print(f"[Worker] Using provided deck_content ({len(deck_content)} chars)")
        # Priority 3: No content found
        else:
            deck_content = "Error: Could not extract text from pitch deck. Please ensure the PDF is readable or contains text."
            print(f"[Worker] Failed to extract content from {deck_path}")
        
        print(f"[Worker] Deck content preview: {deck_content[:200]}...")

        
        # Create all 3 agents
        scribe = create_scribe_agent()
        researcher = create_researcher_agent()
        analyst = create_analyst_agent(investor_context=investor_context)
        
        # Create tasks - full pipeline with web research
        task1 = Task(
            description=f'''Extract key claims from the following pitch deck text.
            Focus on the COMPANY being pitched (ignore sample dashboard data like example store names).
            Extract specific numbers, metrics, market sizes, growth rates, and revenue figures about the company.
            
            PITCH DECK TEXT:
            {deck_content[:4000]}''',
            agent=scribe,
            expected_output='A bulleted list of specific, verifiable claims about the company with numbers and dates.'
        )
        
        task2 = Task(
            description='''Verify the Top 3 most important claims about the company using web search.
            For each claim:
            1. Search for supporting or contradicting evidence
            2. Include the source URLs from search results in your report
            
            Format your verification report like this:
            - Claim: [the claim]
            - Status: CONFIRMED / CONTRADICTED / UNVERIFIED
            - Evidence: [summary of what you found]
            - Source: [the URL from search results]''',
            agent=researcher,
            expected_output='A verification report with status and source URLs for each claim.',
            context=[task1]
        )
        
        task3 = Task(
            description='''Review the extracted claims and verification report critically.
            Focus on the COMPANY being pitched, not sample data or example stores.
            For each major claim, identify:
            1. Red flags or inconsistencies
            2. What information is missing
            3. Key questions to ask the founders
            
            Include a References section at the end with the source URLs from the verification report.
            
            Be skeptical and thorough.''',
            agent=analyst,
            expected_output='A detailed due diligence report with red flags, missing info, questions, and a References section with URLs.',
            context=[task1, task2]
        )
        
        # Create and run crew
        crew = Crew(
            agents=[scribe, researcher, analyst],
            tasks=[task1, task2, task3],
            verbose=True,
            process=Process.sequential,
        )

        
        result = crew.kickoff()
        
        # Extract results
        claims = str(task1.output) if task1.output else ""
        verification = str(task2.output) if task2.output else ""
        report = str(result)
        
        # Update job as completed
        update_job_completed(db, job_id, claims, verification, report)


        print(f"[Worker] Job {job_id} completed successfully")
        
    except Exception as e:
        print(f"[Worker] Job {job_id} failed: {str(e)}")
        update_job_failed(db, job_id, str(e))
    finally:
        db.close()


def process_job(job_data: dict):
    """Process a single job from the queue."""
    job_id = job_data.get("job_id")
    investor_id = job_data.get("investor_id")
    deck_content = job_data.get("deck_content")
    deck_path = job_data.get("deck_path")  # New: file path for PDF
    
    if not job_id:
        print("[Worker] Invalid job data - no job_id")
        return
    
    print(f"[Worker] Processing job - deck_path: {deck_path}, has_content: {bool(deck_content)}")
    run_analysis(job_id, investor_id, deck_content, deck_path)


def main():
    """Main worker loop - listens for jobs on Redis queue."""
    print(f"[Worker] Starting job worker, listening on {JOB_QUEUE}...")
    print(f"[Worker] Redis: {REDIS_URL}")
    
    while True:
        try:
            # Block and wait for job (timeout 0 = wait forever)
            result = redis_client.blpop(JOB_QUEUE, timeout=30)
            
            if result:
                queue_name, job_data = result
                job = json.loads(job_data)
                print(f"[Worker] Received job: {job}")
                process_job(job)
            else:
                # Timeout - just continue waiting
                pass
                
        except redis.ConnectionError as e:
            print(f"[Worker] Redis connection error: {e}")
            time.sleep(5)  # Wait before retry
        except json.JSONDecodeError as e:
            print(f"[Worker] Invalid job JSON: {e}")
        except KeyboardInterrupt:
            print("[Worker] Shutting down...")
            break
        except Exception as e:
            print(f"[Worker] Unexpected error: {e}")
            time.sleep(1)


if __name__ == "__main__":
    main()
