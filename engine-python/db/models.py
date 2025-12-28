"""
Database connection and models for Python engine.
"""
import os
from sqlalchemy import create_engine, Column, String, Text, DateTime, ARRAY
from sqlalchemy.dialects.postgresql import UUID, JSONB
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from sqlalchemy.sql import func
from dotenv import load_dotenv
import uuid

load_dotenv()

# Database connection
DATABASE_URL = os.getenv(
    "DATABASE_URL", 
    "postgresql://sago:sago_dev_password@localhost:5433/sago"
)

engine = create_engine(DATABASE_URL)
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
Base = declarative_base()


class Investor(Base):
    __tablename__ = "investors"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    email = Column(String(255), unique=True, nullable=False)
    name = Column(String(255))
    investment_thesis = Column(Text)
    focus_areas = Column(ARRAY(Text))
    deal_breakers = Column(ARRAY(Text))
    notes = Column(Text)
    created_at = Column(DateTime, server_default=func.now())
    updated_at = Column(DateTime, server_default=func.now())


class PitchDeck(Base):
    __tablename__ = "pitch_decks"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    investor_id = Column(UUID(as_uuid=True))
    filename = Column(String(255), nullable=False)
    gcs_path = Column(Text)
    file_hash = Column(String(64))
    source = Column(String(50), default="upload")
    source_metadata = Column(JSONB)
    created_at = Column(DateTime, server_default=func.now())


class AnalysisJob(Base):
    __tablename__ = "analysis_jobs"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    deck_id = Column(UUID(as_uuid=True))
    investor_id = Column(UUID(as_uuid=True))
    status = Column(String(50), default="pending")
    claims_extracted = Column(JSONB)
    verification_results = Column(JSONB)
    final_report = Column(Text)
    final_report_gcs_path = Column(Text)
    error_message = Column(Text)
    started_at = Column(DateTime)
    completed_at = Column(DateTime)
    created_at = Column(DateTime, server_default=func.now())


def get_db():
    """Get database session."""
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


def get_job_by_id(db, job_id: str) -> AnalysisJob:
    """Retrieve a job by ID."""
    return db.query(AnalysisJob).filter(AnalysisJob.id == job_id).first()


def get_investor_by_id(db, investor_id: str) -> Investor:
    """Retrieve an investor by ID."""
    return db.query(Investor).filter(Investor.id == investor_id).first()


def update_job_status(db, job_id: str, status: str):
    """Update job status."""
    job = get_job_by_id(db, job_id)
    if job:
        job.status = status
        db.commit()


def update_job_started(db, job_id: str):
    """Mark job as started."""
    job = get_job_by_id(db, job_id)
    if job:
        job.status = "running"
        job.started_at = func.now()
        db.commit()


def update_job_completed(db, job_id: str, claims: str, verification: str, report: str):
    """Mark job as completed with results."""
    job = get_job_by_id(db, job_id)
    if job:
        job.status = "completed"
        job.claims_extracted = {"raw": claims}
        job.verification_results = {"raw": verification}
        job.final_report = report
        job.completed_at = func.now()
        db.commit()


def update_job_failed(db, job_id: str, error_msg: str):
    """Mark job as failed."""
    job = get_job_by_id(db, job_id)
    if job:
        job.status = "failed"
        job.error_message = error_msg
        job.completed_at = func.now()
        db.commit()
