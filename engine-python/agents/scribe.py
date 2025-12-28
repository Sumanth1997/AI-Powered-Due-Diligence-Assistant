import os
from crewai import Agent, LLM

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

def create_scribe_agent():
    return Agent(
        role='Scribe',
        goal='Extract specific, verifiable claims from pitch decks',
        backstory=(
            "You are an expert financial analyst and scribe. Your job is to read pitch decks "
            "and extract every specific claim made by the founders. You focus on numbers, "
            "dates, partnership names, and growth metrics. You ignore vague marketing fluff. "
            "You MUST output the list of claims directly as your Final Answer."
        ),
        llm=get_llm(),
        tools=[],
        verbose=True,
        allow_delegation=False
    )
