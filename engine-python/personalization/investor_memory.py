"""
Investor Memory Module
Stores and retrieves investor profiles, preferences, and historical memos using Pinecone.
"""
import os
from typing import List, Dict, Optional
from dotenv import load_dotenv
from pinecone import Pinecone, ServerlessSpec
from sentence_transformers import SentenceTransformer

load_dotenv()

class InvestorMemory:
    def __init__(self):
        # Initialize Pinecone
        self.pc = Pinecone(api_key=os.getenv("PINECONE_API_KEY"))
        self.index_name = os.getenv("PINECONE_INDEX", "sago-investors")
        
        # Initialize embedding model (bge-large outputs 1024 dimensions)
        print("Loading embedding model (this may take a moment on first run)...")
        self.embedder = SentenceTransformer('BAAI/bge-large-en-v1.5')
        print("Embedding model loaded!")
        
        # Ensure index exists
        self._ensure_index()
        
        # Connect to index
        self.index = self.pc.Index(self.index_name)

    
    def _ensure_index(self):
        """Create index if it doesn't exist or dimensions don't match."""
        try:
            existing_indexes = [idx.name for idx in self.pc.list_indexes()]
            if self.index_name in existing_indexes:
                # Index exists, check if we need to delete it
                print(f"Index {self.index_name} exists, connecting...")
                return
            
            # Create new index
            self.pc.create_index(
                name=self.index_name,
                dimension=1024,  # bge-large embedding dimension
                metric="cosine",
                spec=ServerlessSpec(
                    cloud="aws",
                    region=os.getenv("PINECONE_ENV", "us-east-1")
                )
            )
            print(f"Created Pinecone index: {self.index_name}")
        except Exception as e:
            print(f"Index setup note: {e}")

    
    def store_investor_profile(self, investor_id: str, profile: Dict):
        """
        Store an investor's profile including:
        - Investment thesis
        - Deal-breaker criteria
        - Historical memos/notes
        - Focus areas (sectors, stages, geographies)
        """
        # Create text representation for embedding
        text_parts = []
        if "thesis" in profile:
            text_parts.append(f"Investment Thesis: {profile['thesis']}")
        if "deal_breakers" in profile:
            text_parts.append(f"Deal Breakers: {', '.join(profile['deal_breakers'])}")
        if "focus_areas" in profile:
            text_parts.append(f"Focus Areas: {', '.join(profile['focus_areas'])}")
        if "notes" in profile:
            text_parts.append(f"Notes: {profile['notes']}")
        
        full_text = "\n".join(text_parts)
        
        # Generate embedding
        embedding = self.embedder.encode(full_text).tolist()
        
        # Upsert to Pinecone
        self.index.upsert(
            vectors=[{
                "id": f"profile_{investor_id}",
                "values": embedding,
                "metadata": {
                    "investor_id": investor_id,
                    "type": "profile",
                    **profile
                }
            }]
        )
        print(f"Stored profile for investor: {investor_id}")
    
    def store_memo(self, investor_id: str, memo_id: str, memo_text: str, metadata: Dict = None):
        """Store an investment memo or note from past deals."""
        embedding = self.embedder.encode(memo_text).tolist()
        
        meta = {
            "investor_id": investor_id,
            "type": "memo",
            "text": memo_text[:1000],  # Truncate for metadata storage
        }
        if metadata:
            meta.update(metadata)
        
        self.index.upsert(
            vectors=[{
                "id": f"memo_{investor_id}_{memo_id}",
                "values": embedding,
                "metadata": meta
            }]
        )
        print(f"Stored memo {memo_id} for investor: {investor_id}")
    
    def get_relevant_context(self, investor_id: str, query: str, top_k: int = 3) -> List[Dict]:
        """
        Retrieve relevant context from investor's history based on the current pitch deck claims.
        This is used to personalize the Analyst's output.
        """
        # Embed the query (e.g., claims from pitch deck)
        query_embedding = self.embedder.encode(query).tolist()
        
        # Search for relevant context from this investor's data
        results = self.index.query(
            vector=query_embedding,
            top_k=top_k,
            include_metadata=True,
            filter={"investor_id": {"$eq": investor_id}}
        )
        
        contexts = []
        for match in results.matches:
            contexts.append({
                "score": match.score,
                "type": match.metadata.get("type"),
                "content": match.metadata
            })
        
        return contexts
    
    def get_investor_focus(self, investor_id: str) -> Optional[str]:
        """
        Get a formatted string of investor's focus areas and deal-breakers
        for injection into the Analyst agent's prompt.
        """
        try:
            # Fetch the profile directly
            result = self.index.fetch(ids=[f"profile_{investor_id}"])
            
            if f"profile_{investor_id}" in result.vectors:
                meta = result.vectors[f"profile_{investor_id}"].metadata
                
                focus_text = []
                if meta.get("focus_areas"):
                    focus_text.append(f"Focus Areas: {', '.join(meta['focus_areas'])}")
                if meta.get("deal_breakers"):
                    focus_text.append(f"Deal Breakers: {', '.join(meta['deal_breakers'])}")
                if meta.get("thesis"):
                    focus_text.append(f"Investment Thesis: {meta['thesis']}")
                
                return "\n".join(focus_text) if focus_text else None
        except Exception as e:
            print(f"Error fetching investor focus: {e}")
        
        return None


# Convenience function for quick setup
def create_demo_investor():
    """Create a demo investor profile for testing."""
    memory = InvestorMemory()
    
    demo_profile = {
        "thesis": "Focus on B2B SaaS companies with strong recurring revenue and clear path to profitability",
        "deal_breakers": [
            "Burn rate > 3x revenue",
            "No clear competitive moat",
            "Founder-market fit concerns",
            "TAM under $1B"
        ],
        "focus_areas": [
            "B2B SaaS",
            "Enterprise Software",
            "Developer Tools",
            "Series A-B stage"
        ],
        "notes": "Prefer companies with NRR > 120% and gross margins > 70%"
    }
    
    memory.store_investor_profile("demo_investor", demo_profile)
    
    # Store a sample memo
    sample_memo = """
    Passed on XYZ Corp - valuation too high relative to ARR ($50M valuation at $2M ARR = 25x).
    Team was strong but go-to-market strategy unclear. Revisit if they hit $5M ARR.
    """
    memory.store_memo("demo_investor", "memo_001", sample_memo, {"company": "XYZ Corp", "decision": "pass"})
    
    print("Demo investor created successfully!")
    return memory


if __name__ == "__main__":
    # Test the module
    print("Initializing Investor Memory...")
    create_demo_investor()
    
    # Test retrieval
    memory = InvestorMemory()
    focus = memory.get_investor_focus("demo_investor")
    print(f"\nInvestor Focus:\n{focus}")
