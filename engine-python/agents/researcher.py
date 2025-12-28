import os
from crewai import Agent, LLM
from crewai_tools import SerperDevTool

def get_llm():
    """Get the configured LLM (OpenAI or fallback)."""
    openai_key = os.getenv("OPENAI_API_KEY")
    if openai_key and openai_key != 'YOUR_OPENAI_API_KEY_HERE' and openai_key != 'NA':
        return LLM(
            model="openai/gpt-4o-mini",
            api_key=openai_key,
            temperature=0.7
        )
    # Fallback to local Ollama
    return LLM(
        model="ollama/llama3.2",
        base_url="http://localhost:11434"
    )

def create_researcher_agent():
    search_tool = SerperDevTool()
    
    return Agent(
        role='Forensic Researcher',
        goal='Verify claims using web search and include actual source URLs in your findings',
        backstory=(
            "You are an investigative researcher. You search the web to verify claims. "
            "The search tool returns results with 'link' fields containing URLs. "
            "You MUST include these actual URLs (like https://example.com/article) in your report. "
            "Format: Claim, Status (CONFIRMED/CONTRADICTED/UNVERIFIED), Evidence, Source URLs."
        ),
        llm=get_llm(),
        tools=[search_tool],
        verbose=True,
        allow_delegation=False
    )


