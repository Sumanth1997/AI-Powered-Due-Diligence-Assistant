import os
import sys
from dotenv import load_dotenv
import PyPDF2

load_dotenv()

from crewai import Agent, Task, Crew, Process
from agents.scribe import create_scribe_agent
from agents.researcher import create_researcher_agent
from agents.analyst import create_analyst_agent

# Define Tasks
def create_tasks(deck_content, agents):
    scribe = agents['scribe']
    researcher = agents['researcher']
    analyst = agents['analyst']
    
    task1 = Task(
        description=f'''
        Extract key claims from the following text as a PYTHON LIST OF STRINGS.
        Text:
        {deck_content}
        ''',
        agent=scribe,
        expected_output='A simple bulleted list of claims. Do NOT include any intro or outro text. Just the list. DO NOT say "I now can give a great answer".',
        output_file='outputs/claims.txt'
    )

    task2 = Task(
        description='''Verify the Top 3 most important claims from the Scribe's list. Use the Search tool for EACH claim.
        
        After searching, compile your findings into a verification report with this format for each claim:
        - Claim: [the claim]
        - Status: VERIFIED / FALSE / UNVERIFIED  
        - Evidence: [brief summary from search]
        
        Your Final Answer MUST be the complete verification report, not an action.''',
        agent=researcher,
        expected_output='A markdown verification report with status for each of the Top 3 claims.',
        context=[task1],
        output_file='outputs/verification_report.md'
    )

    task3 = Task(
        description='''
        Review the Claims and the Verification Report. 
        Identify:
        1. Confirmed Facts (High confidence)
        2. Red Flags (Contradictions, lack of proof)
        3. Missing Information (Critical business metrics not mentioned)
        
        Generate 5-10 tough questions for the founders.
        
        DO NOT say "I now can give a great answer". Just give the report.
        ''',
        agent=analyst,
        expected_output='A Final Due Diligence Report with Red Flags and Questions.',
        context=[task1, task2],
        output_file='outputs/final_report.md'
    )

    return [task1, task2, task3]

def extract_text_from_pdf(pdf_path):
    with open(pdf_path, 'rb') as file:
        reader = PyPDF2.PdfReader(file)
        text = ""
        for page in reader.pages:
            text += page.extract_text() + "\n"
    return text

# Main execution
if __name__ == "__main__":
    
    # Path to the text file
    deck_path = "../mock_pitch_deck.txt"
    
    if os.path.exists(deck_path):
        print(f"Reading Deck from: {deck_path}")
        with open(deck_path, 'r') as f:
            deck_content = f.read()
    else:
        print(f"Error: Text file not found at {deck_path}")
        # Fallback for testing if file doesn't exist
        deck_content = """
        Startup Name: Sago (Fallback)
        Topic: Due Diligence AI
        """
    
    # Get investor context from Pinecone (if available)
    investor_context = None
    investor_id = os.getenv("INVESTOR_ID", "demo_investor")
    
    try:
        from personalization.investor_memory import InvestorMemory
        memory = InvestorMemory()
        investor_context = memory.get_investor_focus(investor_id)
        if investor_context:
            print(f"\n=== Personalization Loaded for: {investor_id} ===")
            print(investor_context)
            print("=" * 50 + "\n")
    except Exception as e:
        print(f"Note: Investor personalization not available: {e}")
    
    scribe = create_scribe_agent()
    researcher = create_researcher_agent()
    analyst = create_analyst_agent(investor_context=investor_context)
    
    agents = {
        'scribe': scribe,
        'researcher': researcher,
        'analyst': analyst
    }
    
    tasks = create_tasks(deck_content, agents)
    
    crew = Crew(
        agents=[scribe, researcher, analyst],
        tasks=tasks,
        verbose=True,
        process=Process.sequential
    )
    
    result = crew.kickoff()
    print("######################")
    print("## Final Report")
    print(result)
