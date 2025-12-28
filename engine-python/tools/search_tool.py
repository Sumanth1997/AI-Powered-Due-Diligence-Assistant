"""
Custom Search Tool with explicit URL extraction
Wraps SerperDevTool to format results with clear URL citations
"""
import os
from crewai.tools import BaseTool
from pydantic import Field
import requests


class SearchWithCitations(BaseTool):
    """Search tool that returns results with explicit URLs."""
    
    name: str = "Web Search with Citations"
    description: str = (
        "Search the internet and get results WITH SOURCE URLs. "
        "Use this to verify claims - it returns the actual webpage links you can cite."
    )
    
    def _run(self, query: str) -> str:
        """Execute search and return formatted results with URLs."""
        api_key = os.getenv("SERPER_API_KEY")
        if not api_key:
            return "Error: SERPER_API_KEY not set"
        
        try:
            response = requests.post(
                "https://google.serper.dev/search",
                headers={
                    "X-API-KEY": api_key,
                    "Content-Type": "application/json"
                },
                json={"q": query, "num": 5}
            )
            data = response.json()
            
            results = []
            organic = data.get("organic", [])
            
            for i, item in enumerate(organic[:5], 1):
                title = item.get("title", "No title")
                link = item.get("link", "No URL")
                snippet = item.get("snippet", "No description")
                
                results.append(
                    f"**Result {i}:**\n"
                    f"  Title: {title}\n"
                    f"  URL: {link}\n"
                    f"  Summary: {snippet}\n"
                )
            
            if not results:
                return f"No search results found for: {query}"
            
            output = f"=== SEARCH RESULTS FOR: {query} ===\n\n"
            output += "\n".join(results)
            output += "\n\n=== USE THE URLs ABOVE AS CITATIONS IN YOUR REPORT ==="
            
            return output
            
        except Exception as e:
            return f"Search failed: {str(e)}"


def create_search_tool():
    """Create the custom search tool instance."""
    return SearchWithCitations()
