import os
from typing import Optional
from functools import lru_cache


class Settings:
    
    PORT: int = int(os.getenv("PORT", "8000"))
    HOST: str = os.getenv("HOST", "0.0.0.0")
    ENVIRONMENT: str = os.getenv("ENVIRONMENT", "development")
    DEBUG: bool = os.getenv("DEBUG", "true").lower() == "true"
    
    
    MODEL_PATH: str = os.getenv("MODEL_PATH", "./models")
    MODEL_VERSION: str = os.getenv("MODEL_VERSION", "v1.2.0")
    BATCH_SIZE: int = int(os.getenv("BATCH_SIZE", "32"))
    SEQUENCE_LENGTH: int = int(os.getenv("SEQUENCE_LENGTH", "10"))
    
    
    RETRAIN_INTERVAL_HOURS: int = int(os.getenv("RETRAIN_INTERVAL_HOURS", "168")) 
    LEARNING_RATE: float = float(os.getenv("LEARNING_RATE", "0.001"))
    EPOCHS: int = int(os.getenv("EPOCHS", "100"))
    EARLY_STOPPING_PATIENCE: int = int(os.getenv("EARLY_STOPPING_PATIENCE", "10"))
    
    
    DATABASE_URL: str = os.getenv("DATABASE_URL", "postgresql://kanopt:kanopt@localhost:5432/kanopt")
    
    
    REDIS_URL: str = os.getenv("REDIS_URL", "redis://localhost:6379")
    CACHE_TTL: int = int(os.getenv("CACHE_TTL", "3600"))  
    
    
    KANBAN_API_URL: str = os.getenv("KANBAN_API_URL", "http://localhost:8080")
    
    
    API_KEY: Optional[str] = os.getenv("API_KEY")
    ALLOWED_ORIGINS: list = os.getenv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001").split(",")
    
    
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")
    LOG_FORMAT: str = os.getenv("LOG_FORMAT", "json")
    
    
    ENABLE_RETRAINING: bool = os.getenv("ENABLE_RETRAINING", "true").lower() == "true"
    ENABLE_RISK_ANALYSIS: bool = os.getenv("ENABLE_RISK_ANALYSIS", "true").lower() == "true"
    ENABLE_METRICS: bool = os.getenv("ENABLE_METRICS", "true").lower() == "true"


@lru_cache()
def get_settings() -> Settings:
    """Get cached settings instance."""
    return Settings()
