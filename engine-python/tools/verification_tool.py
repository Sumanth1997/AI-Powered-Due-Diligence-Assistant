import os
from crewai_tools import SerperDevTool
from crewai.tools import BaseTool
from openai import OpenAI
from dotenv import load_dotenv

load_dotenv()

# Shared log file path
VERIFICATION_LOG = "outputs/verification_log.md"

class ClaimVerifierTool(BaseTool):
    name: str = "Verify Claim"
    description: str = "Verifies a SINGLE claim using web search and returns a concise YES/NO summary. Input should be the claim string."

    def _run(self, claim: str) -> str:
        # 1. Search (using existing SerperDevTool logic)
        search_tool = SerperDevTool(n_results=3)
        try:
            search_result = search_tool.run(search_query=claim)
        except Exception as e:
            return f"Error during search: {str(e)}"

        # 2. Synthesize using separate LLM call (The "Chunking" trick)
        try:
            client = OpenAI(
                base_url=os.getenv("OPENAI_API_BASE"),
                api_key=os.getenv("OPENAI_API_KEY")
            )
            
            prompt = f"""
            You are a strict fact checker. 
            
            Claim: "{claim}"
            
            Evidence from Search:
            {search_result}
            
            Task: Verify the claim based on the evidence.
            Output format:
            - Status: [VERIFIED / FALSE / UNVERIFIED]
            - Explanation: [Concise 1-sentence explanation citing the source if available]
            """

            response = client.chat.completions.create(
                model=os.getenv("OPENAI_MODEL_NAME"),
                messages=[
                    {"role": "system", "content": "You are a helpful assistant."},
                    {"role": "user", "content": prompt}
                ]
            )
            
            result = response.choices[0].message.content
            
            # 3. Log result to file (persistent record)
            self._log_verification(claim, result)
            
            return result
            
        except Exception as e:
            return f"Error during LLM synthesis: {str(e)}"

    def _log_verification(self, claim: str, result: str):
        """Append verification result to shared log file."""
        try:
            os.makedirs(os.path.dirname(VERIFICATION_LOG), exist_ok=True)
            with open(VERIFICATION_LOG, "a") as f:
                f.write(f"## Claim: {claim}\n")
                f.write(f"{result}\n\n---\n\n")
        except Exception:
            pass  # Non-critical, don't break the flow
