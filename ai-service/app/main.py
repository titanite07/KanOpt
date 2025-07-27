from fastapi import FastAPI, HTTPException, Depends, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
import uvicorn
import asyncio
from contextlib import asynccontextmanager
from loguru import logger

from app.config import get_settings
from app.models.predictor import VelocityPredictor
from app.models.risk_analyzer import RiskAnalyzer
from app.services.data_service import DataService
from app.services.training_service import TrainingService
from app.api.routes import router
from app.middleware.auth import auth_middleware
from app.middleware.metrics import metrics_middleware

settings = get_settings()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager."""
    logger.info("🚀 Starting AI Service")
    
    # Initialize services
    app.state.data_service = DataService()
    app.state.velocity_predictor = VelocityPredictor()
    app.state.risk_analyzer = RiskAnalyzer()
    app.state.training_service = TrainingService()
    
    # Load models
    try:
        await app.state.velocity_predictor.load_model()
        await app.state.risk_analyzer.load_model()
        logger.info("✅ Models loaded successfully")
    except Exception as e:
        logger.warning(f"⚠️ Failed to load models: {e}")
        logger.info("🔄 Will train new models on first request")
    
    # Start background training scheduler
    if settings.ENABLE_RETRAINING:
        training_task = asyncio.create_task(
            app.state.training_service.schedule_retraining()
        )
        logger.info("📅 Background training scheduler started")
    
    yield
    
    # Cleanup
    logger.info("🛑 Shutting down AI Service")
    if settings.ENABLE_RETRAINING:
        training_task.cancel()
        try:
            await training_task
        except asyncio.CancelledError:
            pass
    
    # Save models before shutdown
    try:
        await app.state.velocity_predictor.save_model()
        await app.state.risk_analyzer.save_model()
        logger.info("✅ Models saved successfully")
    except Exception as e:
        logger.error(f"❌ Failed to save models: {e}")


def create_app() -> FastAPI:
    """Create FastAPI application."""
    
    app = FastAPI(
        title="KanOpt AI Service",
        description="LSTM-based velocity prediction and risk analysis for Kanban boards",
        version=settings.MODEL_VERSION,
        lifespan=lifespan,
        docs_url="/docs" if settings.DEBUG else None,
        redoc_url="/redoc" if settings.DEBUG else None,
    )
    
    # CORS middleware
    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.ALLOWED_ORIGINS,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )
    
    # Custom middleware
    if settings.API_KEY:
        app.middleware("http")(auth_middleware)
    
    if settings.ENABLE_METRICS:
        app.middleware("http")(metrics_middleware)
    
    # Include API routes
    app.include_router(router, prefix="/api")
    
    # Health check endpoint
    @app.get("/health")
    async def health_check():
        """Health check endpoint."""
        return {
            "status": "healthy",
            "version": settings.MODEL_VERSION,
            "environment": settings.ENVIRONMENT,
            "features": {
                "velocity_prediction": True,
                "risk_analysis": settings.ENABLE_RISK_ANALYSIS,
                "auto_retraining": settings.ENABLE_RETRAINING,
                "metrics": settings.ENABLE_METRICS,
            }
        }
    
    # Model info endpoint
    @app.get("/api/model/info")
    async def model_info():
        """Get model information."""
        try:
            velocity_info = await app.state.velocity_predictor.get_model_info()
            risk_info = await app.state.risk_analyzer.get_model_info()
            
            return {
                "velocity_model": velocity_info,
                "risk_model": risk_info,
                "version": settings.MODEL_VERSION,
                "last_training": None,  
                "next_training": None,  
            }
        except Exception as e:
            logger.error(f"Failed to get model info: {e}")
            raise HTTPException(status_code=500, detail="Failed to get model information")
    
    # Global exception handler
    @app.exception_handler(Exception)
    async def global_exception_handler(request, exc):
        logger.error(f"Unhandled exception: {exc}")
        return JSONResponse(
            status_code=500,
            content={"detail": "Internal server error"}
        )
    
    return app


app = create_app()


if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        log_level=settings.LOG_LEVEL.lower(),
    )
