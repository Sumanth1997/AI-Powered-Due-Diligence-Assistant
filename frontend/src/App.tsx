import { useState, useEffect, useCallback } from 'react';
import ReactMarkdown from 'react-markdown';
import './App.css';
import * as api from './api';
import type { Job, Investor } from './api';

type Tab = 'analysis' | 'investors' | 'gmail';

function App() {
  const [activeTab, setActiveTab] = useState<Tab>('analysis');
  const [jobs, setJobs] = useState<Job[]>([]);
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [uploading, setUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const [jobIds, setJobIds] = useState<string[]>([]);

  // Investors state
  const [investors, setInvestors] = useState<Investor[]>([]);
  const [selectedInvestor, setSelectedInvestor] = useState<Investor | null>(null);
  const [showNewInvestor, setShowNewInvestor] = useState(false);
  const [newInvestor, setNewInvestor] = useState({
    email: '',
    name: '',
    investment_thesis: '',
    focus_areas: [] as string[],
    deal_breakers: [] as string[],
  });
  const [focusTag, setFocusTag] = useState('');
  const [dealBreakerTag, setDealBreakerTag] = useState('');

  // Gmail state
  const [gmailDecks, setGmailDecks] = useState<any[]>([]);
  const [loadingGmail, setLoadingGmail] = useState(false);



  // Poll for job updates
  useEffect(() => {
    if (jobIds.length === 0) return;

    const pollJobs = async () => {
      const updatedJobs: Job[] = [];
      for (const id of jobIds) {
        try {
          const job = await api.getJob(id);
          updatedJobs.push(job);
        } catch (error) {
          console.error(`Failed to fetch job ${id}:`, error);
        }
      }
      setJobs(updatedJobs);
    };

    pollJobs();
    const interval = setInterval(pollJobs, 3000);
    return () => clearInterval(interval);
  }, [jobIds]);

  // Update selected job when jobs list updates
  useEffect(() => {
    if (selectedJob) {
      const updated = jobs.find(j => j.id === selectedJob.id);
      if (updated && updated.status !== selectedJob.status) {
        setSelectedJob(updated);
      }
    }
  }, [jobs, selectedJob]);

  // Handle file upload
  const handleUpload = useCallback(async (file: File) => {
    setUploading(true);
    try {
      const investorId = selectedInvestor?.id;
      const response = await api.uploadDeck(file, investorId);
      setJobIds(prev => [response.job_id, ...prev]);

      const job = await api.getJob(response.job_id);
      setJobs(prev => [job, ...prev]);
      setSelectedJob(job);
    } catch (error) {
      console.error('Upload failed:', error);
    } finally {
      setUploading(false);
    }
  }, [selectedInvestor]);

  // Gmail functions
  const checkGmail = async () => {
    setLoadingGmail(true);
    try {
      const data = await api.checkGmailDecks();
      setGmailDecks(data.decks || []);
    } catch (error) {
      console.error('Gmail check failed:', error);
    } finally {
      setLoadingGmail(false);
    }
  };

  const processGmailDeck = async (messageId: string) => {
    try {
      const response = await api.processGmailMessage(messageId);

      // Add the new job to our list
      if (response.job_id) {
        setJobIds(prev => [response.job_id, ...prev]);

        // Fetch the job
        const job = await api.getJob(response.job_id);
        setJobs(prev => [job, ...prev]);
        setSelectedJob(job);

        // Switch to Analysis tab to show progress
        setActiveTab('analysis');
      }

      checkGmail();
    } catch (error) {
      console.error('Gmail process failed:', error);
    }
  };

  // Investor functions
  const createInvestor = async () => {
    try {
      const investor = await api.createInvestor(newInvestor);
      setInvestors(prev => [...prev, investor]);
      setSelectedInvestor(investor);
      setShowNewInvestor(false);
      setNewInvestor({
        email: '',
        name: '',
        investment_thesis: '',
        focus_areas: [],
        deal_breakers: [],
      });
    } catch (error) {
      console.error('Create investor failed:', error);
    }
  };

  const addFocusTag = () => {
    if (focusTag.trim()) {
      setNewInvestor(prev => ({
        ...prev,
        focus_areas: [...prev.focus_areas, focusTag.trim()],
      }));
      setFocusTag('');
    }
  };

  const addDealBreakerTag = () => {
    if (dealBreakerTag.trim()) {
      setNewInvestor(prev => ({
        ...prev,
        deal_breakers: [...prev.deal_breakers, dealBreakerTag.trim()],
      }));
      setDealBreakerTag('');
    }
  };

  const removeTag = (field: 'focus_areas' | 'deal_breakers', index: number) => {
    setNewInvestor(prev => ({
      ...prev,
      [field]: prev[field].filter((_, i) => i !== index),
    }));
  };

  // Drag and drop
  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = () => setDragOver(false);

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files[0];
    if (file) handleUpload(file);
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) handleUpload(file);
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div className="app">
      {/* Header */}
      <header className="header">
        <div className="logo">
          <div className="logo-icon">🔍</div>
          <h1>Sago</h1>
        </div>

        <div className="nav-tabs">
          <button
            className={`nav-tab ${activeTab === 'analysis' ? 'active' : ''}`}
            onClick={() => setActiveTab('analysis')}
          >
            Analysis
          </button>
          <button
            className={`nav-tab ${activeTab === 'investors' ? 'active' : ''}`}
            onClick={() => setActiveTab('investors')}
          >
            Investors
          </button>
          <button
            className={`nav-tab ${activeTab === 'gmail' ? 'active' : ''}`}
            onClick={() => { setActiveTab('gmail'); checkGmail(); }}
          >
            Gmail
          </button>
        </div>


      </header>

      {/* Analysis Tab */}
      {activeTab === 'analysis' && (
        <div className="main-grid">
          <div>
            {/* Upload Card */}
            <div className="card" style={{ marginBottom: '1.5rem' }}>
              <div className="card-header">
                <h2 className="card-title">Upload Pitch Deck</h2>
                {selectedInvestor && (
                  <span className="tag">{selectedInvestor.name || selectedInvestor.email}</span>
                )}
              </div>
              <div
                className={`upload-zone ${dragOver ? 'dragover' : ''}`}
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onDrop={handleDrop}
                onClick={() => document.getElementById('file-input')?.click()}
              >
                <input
                  id="file-input"
                  type="file"
                  accept=".pdf,.txt"
                  onChange={handleFileSelect}
                  style={{ display: 'none' }}
                />
                {uploading ? (
                  <>
                    <div className="spinner" style={{ margin: '0 auto 1rem' }}></div>
                    <p className="upload-text">Uploading & queuing...</p>
                  </>
                ) : (
                  <>
                    <div className="upload-icon">📄</div>
                    <p className="upload-text">
                      <strong>Click to upload</strong> or drag and drop
                    </p>
                    <p className="upload-hint">PDF or TXT • Max 10MB</p>
                  </>
                )}
              </div>
            </div>

            {/* Jobs List */}
            <div className="card">
              <div className="card-header">
                <h2 className="card-title">Recent Jobs</h2>
                <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                  {jobs.length} total
                </span>
              </div>

              {jobs.length === 0 ? (
                <div className="empty-state">
                  <div className="empty-state-icon">📋</div>
                  <p>No analysis jobs yet</p>
                </div>
              ) : (
                <div className="jobs-list">
                  {jobs.map(job => (
                    <div
                      key={job.id}
                      className={`job-item ${selectedJob?.id === job.id ? 'active' : ''}`}
                      onClick={() => setSelectedJob(job)}
                    >
                      <div className="job-info">
                        <span className="job-filename">
                          Job {job.id.slice(0, 8)}
                        </span>
                        <div className="job-meta">
                          <span>{formatDate(job.created_at)}</span>
                          {job.investor_id && <span>• Personalized</span>}
                        </div>
                      </div>
                      <span className={`job-status ${job.status}`}>
                        {job.status}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Report View */}
          <div className="card report-container">
            {!selectedJob ? (
              <div className="report-empty">
                <div className="report-empty-icon">📊</div>
                <p>Select a job to view the report</p>
                <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', marginTop: '0.5rem' }}>
                  Upload a pitch deck to get started
                </p>
              </div>
            ) : selectedJob.status === 'pending' || selectedJob.status === 'running' ? (
              <div className="report-empty">
                <div className="spinner"></div>
                <p style={{ marginTop: '1.5rem' }}>Analysis in progress...</p>
                <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', marginTop: '0.5rem' }}>
                  Scribe → Researcher → Analyst
                </p>
              </div>
            ) : selectedJob.status === 'failed' ? (
              <div className="report-empty">
                <div className="report-empty-icon">❌</div>
                <p>Analysis failed</p>
                <p style={{ fontSize: '0.875rem', color: 'var(--error)', marginTop: '0.5rem' }}>
                  {selectedJob.error_message || 'Unknown error'}
                </p>
              </div>
            ) : (
              <div className="report-content">
                <div className="report-header">
                  <h2 className="report-title">Due Diligence Report</h2>
                  <span className={`job-status ${selectedJob.status}`}>
                    {selectedJob.status}
                  </span>
                </div>
                <div className="report-body markdown-content">
                  {selectedJob.final_report ? (
                    <ReactMarkdown>{selectedJob.final_report}</ReactMarkdown>
                  ) : (
                    <p style={{ color: 'var(--text-muted)' }}>No report content available</p>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Investors Tab */}
      {activeTab === 'investors' && (
        <div className="investor-grid">
          {/* Investors List */}
          <div className="card">
            <div className="card-header">
              <h2 className="card-title">Investor Profiles</h2>
              <button className="btn btn-primary" onClick={() => setShowNewInvestor(true)}>
                + New
              </button>
            </div>

            {investors.length === 0 && !showNewInvestor ? (
              <div className="empty-state">
                <div className="empty-state-icon">👤</div>
                <p>No investor profiles yet</p>
                <button
                  className="btn btn-secondary"
                  style={{ marginTop: '1rem' }}
                  onClick={() => setShowNewInvestor(true)}
                >
                  Create First Profile
                </button>
              </div>
            ) : (
              <div className="investor-list">
                {investors.map(investor => (
                  <div key={investor.id}
                    className={`investor-item ${selectedInvestor?.id === investor.id ? 'active' : ''}`}
                    onClick={() => setSelectedInvestor(investor)}
                  >
                    <div className="investor-name">{investor.name?.trim() || investor.email}</div>
                    <div className="investor-email">{investor.email}</div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Investor Form / Details */}
          <div className="card">
            {showNewInvestor ? (
              <>
                <div className="card-header">
                  <h2 className="card-title">New Investor</h2>
                  <button className="btn-icon" onClick={() => setShowNewInvestor(false)}>✕</button>
                </div>

                <div className="form-group">
                  <label className="form-label">Email</label>
                  <input
                    type="email"
                    className="form-input"
                    placeholder="investor@example.com"
                    value={newInvestor.email}
                    onChange={e => setNewInvestor(prev => ({ ...prev, email: e.target.value }))}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Name</label>
                  <input
                    type="text"
                    className="form-input"
                    placeholder="John Doe"
                    value={newInvestor.name}
                    onChange={e => setNewInvestor(prev => ({ ...prev, name: e.target.value }))}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Investment Thesis</label>
                  <textarea
                    className="form-textarea"
                    placeholder="Focus on B2B SaaS with strong unit economics..."
                    value={newInvestor.investment_thesis}
                    onChange={e => setNewInvestor(prev => ({ ...prev, investment_thesis: e.target.value }))}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Focus Areas</label>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <input
                      type="text"
                      className="form-input"
                      placeholder="e.g., B2B SaaS"
                      value={focusTag}
                      onChange={e => setFocusTag(e.target.value)}
                      onKeyDown={e => e.key === 'Enter' && addFocusTag()}
                    />
                    <button className="btn btn-secondary" onClick={() => addFocusTag()}>Add</button>
                  </div>
                  <div className="tags-container">
                    {newInvestor.focus_areas.map((tag, i) => (
                      <span key={i} className="tag">
                        {tag}
                        <span className="tag-remove" onClick={() => removeTag('focus_areas', i)}>✕</span>
                      </span>
                    ))}
                  </div>
                </div>

                <div className="form-group">
                  <label className="form-label">Deal Breakers</label>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <input
                      type="text"
                      className="form-input"
                      placeholder="e.g., Burn > 3x revenue"
                      value={dealBreakerTag}
                      onChange={e => setDealBreakerTag(e.target.value)}
                      onKeyDown={e => e.key === 'Enter' && addDealBreakerTag()}
                    />
                    <button className="btn btn-secondary" onClick={() => addDealBreakerTag()}>Add</button>
                  </div>
                  <div className="tags-container">
                    {newInvestor.deal_breakers.map((tag, i) => (
                      <span key={i} className="tag" style={{ background: 'rgba(239, 68, 68, 0.15)', color: 'var(--error)' }}>
                        {tag}
                        <span className="tag-remove" onClick={() => removeTag('deal_breakers', i)}>✕</span>
                      </span>
                    ))}
                  </div>
                </div>

                <button
                  className="btn btn-primary"
                  style={{ width: '100%', marginTop: '1rem' }}
                  onClick={createInvestor}
                  disabled={!newInvestor.email}
                >
                  Create Investor
                </button>
              </>
            ) : selectedInvestor ? (
              <>
                <div className="card-header">
                  <h2 className="card-title">Investor Details</h2>
                </div>
                <div style={{ marginBottom: '1rem' }}>
                  <h3 style={{ fontSize: '1.25rem', marginBottom: '0.25rem' }}>{selectedInvestor.name}</h3>
                  <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>{selectedInvestor.email}</p>
                </div>

                {selectedInvestor.investment_thesis && (
                  <div style={{ marginBottom: '1rem' }}>
                    <label className="form-label">Thesis</label>
                    <p style={{ color: 'var(--text-secondary)' }}>{selectedInvestor.investment_thesis}</p>
                  </div>
                )}

                {selectedInvestor.focus_areas && selectedInvestor.focus_areas.length > 0 && (
                  <div style={{ marginBottom: '1rem' }}>
                    <label className="form-label">Focus Areas</label>
                    <div className="tags-container">
                      {selectedInvestor.focus_areas.map((tag, i) => (
                        <span key={i} className="tag">{tag}</span>
                      ))}
                    </div>
                  </div>
                )}

                {selectedInvestor.deal_breakers && selectedInvestor.deal_breakers.length > 0 && (
                  <div>
                    <label className="form-label">Deal Breakers</label>
                    <div className="tags-container">
                      {selectedInvestor.deal_breakers.map((tag, i) => (
                        <span key={i} className="tag" style={{ background: 'rgba(239, 68, 68, 0.15)', color: 'var(--error)' }}>{tag}</span>
                      ))}
                    </div>
                  </div>
                )}

                <button
                  className="btn btn-primary"
                  style={{ width: '100%', marginTop: '2rem' }}
                  onClick={() => setActiveTab('analysis')}
                >
                  Use for Analysis →
                </button>
              </>
            ) : (
              <div className="report-empty">
                <div className="report-empty-icon">👤</div>
                <p>Select an investor or create new</p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Gmail Tab */}
      {activeTab === 'gmail' && (
        <div className="card">
          <div className="card-header">
            <h2 className="card-title">Gmail Pitch Decks</h2>
            <button className="btn btn-secondary" onClick={checkGmail} disabled={loadingGmail}>
              {loadingGmail ? 'Checking...' : '🔄 Refresh'}
            </button>
          </div>

          {loadingGmail ? (
            <div className="report-empty">
              <div className="spinner"></div>
              <p style={{ marginTop: '1rem' }}>Scanning inbox...</p>
            </div>
          ) : gmailDecks.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state-icon">📧</div>
              <p>No pitch decks found in Gmail</p>
              <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', marginTop: '0.5rem' }}>
                Send yourself an email with "Pitch Deck" in the subject and a PDF attachment
              </p>
            </div>
          ) : (
            <div className="jobs-list">
              {gmailDecks.map(deck => (
                <div key={deck.message_id} className="gmail-deck">
                  <div className="gmail-deck-header">
                    <div>
                      <div className="gmail-deck-subject">{deck.subject}</div>
                      <div className="gmail-deck-sender">{deck.sender}</div>
                    </div>
                    <button
                      className="btn btn-primary"
                      onClick={() => processGmailDeck(deck.message_id)}
                    >
                      Analyze
                    </button>
                  </div>
                  <div className="tags-container">
                    {deck.pdfs?.map((pdf: string, i: number) => (
                      <span key={i} className="tag">📄 {pdf}</span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default App;
