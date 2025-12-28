import os
from crewai import Agent, LLM
from typing import Optional

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

def create_analyst_agent(investor_context: Optional[str] = None):
    base_backstory = (
        "You are a cynical, detail-oriented venture capital analyst. You take the claims "
        "extracted by the Scribe and the verification report from the Researcher, and you "
        "tear them apart. You look for inconsistencies, what's NOT said, and what is too "
        "good to be true. Your output is a brutally honest memo. "
        "You MUST output the memo directly as your Final Answer."
    )
    
    if investor_context:
        personalized_backstory = (
            f"{base_backstory}\n\n"
            f"IMPORTANT - This analysis is for an investor with the following preferences:\n"
            f"{investor_context}\n\n"
            f"Pay special attention to their deal-breakers and focus areas when generating "
            f"red flags and questions. Tailor your analysis to their investment thesis."
        )
    else:
        personalized_backstory = base_backstory
    
    return Agent(
        role='Adversarial Analyst',
        goal='Identify red flags, missing information, and generate key questions tailored to the investor',
        backstory=personalized_backstory,
        llm=get_llm(),
        tools=[],
        verbose=True,
        allow_delegation=False
    )
