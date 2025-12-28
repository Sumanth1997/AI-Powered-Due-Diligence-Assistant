const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export interface Investor {
    id: string;
    email: string;
    name?: string;
    investment_thesis?: string;
    focus_areas?: string[];
    deal_breakers?: string[];
    created_at: string;
}

export interface PitchDeck {
    id: string;
    investor_id?: string;
    filename: string;
    gcs_path?: string;
    source: string;
    created_at: string;
}

export interface Job {
    id: string;
    deck_id?: string;
    investor_id?: string;
    status: 'pending' | 'running' | 'completed' | 'failed';
    claims_extracted?: string;
    verification_results?: string;
    final_report?: string;
    error_message?: string;
    started_at?: string;
    completed_at?: string;
    created_at: string;
}

export interface HealthStatus {
    status: string;
    database: string;
    redis: string;
    gcs: string;
    gmail: string;
}

export interface UploadResponse {
    deck: PitchDeck;
    job_id: string;
    status: string;
}

// Health check
export async function getHealth(): Promise<HealthStatus> {
    const response = await fetch(`${API_BASE}/health`);
    return response.json();
}

// Investors
export async function getInvestors(): Promise<Investor[]> {
    const response = await fetch(`${API_BASE}/investors`);
    if (!response.ok) return [];
    return response.json();
}

export async function getInvestor(id: string): Promise<Investor> {
    const response = await fetch(`${API_BASE}/investors/${id}`);
    return response.json();
}

export async function createInvestor(data: Partial<Investor>): Promise<Investor> {
    const response = await fetch(`${API_BASE}/investors`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
    });
    return response.json();
}

// Decks & Jobs
export async function uploadDeck(file: File, investorId?: string): Promise<UploadResponse> {
    const formData = new FormData();
    formData.append('file', file);
    if (investorId) {
        formData.append('investor_id', investorId);
    }

    const response = await fetch(`${API_BASE}/decks/upload`, {
        method: 'POST',
        body: formData,
    });
    return response.json();
}

export async function getJob(id: string): Promise<Job> {
    const response = await fetch(`${API_BASE}/jobs/${id}`);
    return response.json();
}

// Gmail
export async function checkGmailDecks(): Promise<{ count: number; decks: any[] }> {
    const response = await fetch(`${API_BASE}/gmail/check`);
    return response.json();
}

export async function processGmailMessage(messageId: string): Promise<any> {
    const response = await fetch(`${API_BASE}/gmail/process/${messageId}`, {
        method: 'POST',
    });
    return response.json();
}
