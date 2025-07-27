from fastapi import APIRouter, HTTPException, Depends, BackgroundTasks, Request
from typing import List, Dict, Any, Optional
from pydantic import BaseModel
import asyncio
from loguru import logger

from app.config import get_settings

settings = get_settings()
router = APIRouter()


# Request/Response models
class PredictionRequest(BaseModel):
    boardId: str
    timeHorizon: str = "2weeks"
    metrics: List[str] = ["velocity", "completion", "risk"]
    velocityHistory: List[Dict[str, Any]]
    currentTasks: List[Dict[str, Any]]


class PredictionResponse(BaseModel):
    boardId: str
    timeHorizon: str
    predictions: Dict[str, Any]
    confidence: float
    generatedAt: str
    modelVersion: str


class RiskAnalysisRequest(BaseModel):
    boardId: str
    tasks: List[Dict[str, Any]]
    factors: List[str] = ["deadline", "complexity", "assignee_workload"]


class RiskAnalysisResponse(BaseModel):
    boardId: str
    risks: List[Dict[str, Any]]
    summary: Dict[str, Any]
    recommendations: List[str]


class TrainingRequest(BaseModel):
    boardId: Optional[str] = None
    velocityData: Optional[List[Dict[str, Any]]] = None
    tasksData: Optional[List[Dict[str, Any]]] = None
    forceRetrain: bool = False


# Dependency to get services from app state
def get_velocity_predictor(request: Request):
    return request.app.state.velocity_predictor


def get_risk_analyzer(request: Request):
    return request.app.state.risk_analyzer


def get_training_service(request: Request):
    return request.app.state.training_service


@router.post("/predict", response_model=PredictionResponse)
async def predict_velocity(
    request: PredictionRequest,
    velocity_predictor=Depends(get_velocity_predictor)
):
    # Make prediction
    prediction_result = await velocity_predictor.predict(
        recent_data=request.velocityHistory,
        time_horizon=request.timeHorizon
    )

    # Format response
    predictions = {
        "velocity": {
            "predicted": prediction_result["predicted_velocity"],
            "confidence": prediction_result["confidence"],
            "range": prediction_result["range"],
        }
    }

    # Add completion predictions if requested
    if "completion" in request.metrics:
        total_tasks = len([task for task in request.currentTasks if not task.get('completedAt')])
        predicted_completion = min(total_tasks, int(prediction_result["predicted_velocity"] * 2))

        predictions["completion"] = {
            "expectedTasks": predicted_completion,
            "totalTasks": total_tasks,
            "completionRate": predicted_completion / max(total_tasks, 1),
        }

    # Add basic risk prediction if requested
    if "risk" in request.metrics:
        high_priority_tasks = len([task for task in request.currentTasks if task.get('priority') == 'high'])
        risk_score = min(1.0, high_priority_tasks / max(len(request.currentTasks), 1))

        predictions["risk"] = {
            "overallRisk": "high" if risk_score > 0.7 else "medium" if risk_score > 0.3 else "low",
            "riskScore": risk_score,
            "highPriorityTasks": high_priority_tasks,
        }

    return PredictionResponse(
        boardId=request.boardId,
        timeHorizon=request.timeHorizon,
        predictions=predictions,
        confidence=prediction_result["confidence"],
        generatedAt=prediction_result["prediction_date"],
        modelVersion=settings.MODEL_VERSION
    )


@router.post("/analyze-risk", response_model=RiskAnalysisResponse)
async def analyze_risk(
    request: RiskAnalysisRequest,
    risk_analyzer=Depends(get_risk_analyzer)
):
    # Perform risk analysis
    analysis_result = await risk_analyzer.analyze_risk(
        tasks=request.tasks,
        board_context={}  # Could be enhanced with board-specific context
    )

    # Generate recommendations based on risk analysis
    recommendations = []
    summary = analysis_result["summary"]

    if summary["high_risk_count"] > 0:
        recommendations.append(f"Immediate attention needed for {summary['high_risk_count']} high-risk tasks")

    if summary["average_risk_score"] > 0.6:
        recommendations.append("Overall board risk is elevated - consider workload redistribution")

    if summary["risk_distribution"]["high"] > 0.2:
        recommendations.append("High percentage of risky tasks - review sprint planning")

    return RiskAnalysisResponse(
        boardId=request.boardId,
        risks=analysis_result["risks"],
        summary=analysis_result["summary"],
        recommendations=recommendations
    )


@router.post("/train")
async def train_models(
    request: TrainingRequest,
    background_tasks: BackgroundTasks,
    velocity_predictor=Depends(get_velocity_predictor),
    risk_analyzer=Depends(get_risk_analyzer)
):
    # Start training in background
    async def train_models_background():
        results = {}

        # Train velocity predictor if data provided
        if request.velocityData:
            try:
                velocity_metrics = await velocity_predictor.train(
                    velocity_data=request.velocityData,
                    board_id=request.boardId
                )
                results["velocity_model"] = velocity_metrics
            except Exception as e:
                results["velocity_model"] = {"error": str(e)}

        # Train risk analyzer if data provided
        if request.tasksData:
            try:
                risk_metrics = await risk_analyzer.train(
                    tasks_data=request.tasksData,
                    board_id=request.boardId
                )
                results["risk_model"] = risk_metrics
            except Exception as e:
                results["risk_model"] = {"error": str(e)}

        return results

    # Add to background tasks
    background_tasks.add_task(train_models_background)

    return {
        "message": "Training started in background",
        "boardId": request.boardId,
        "status": "initiated"
    }


@router.get("/model/status")
async def get_model_status(
    velocity_predictor=Depends(get_velocity_predictor),
    risk_analyzer=Depends(get_risk_analyzer)
):
    # Get current model status and information
    velocity_info = await velocity_predictor.get_model_info()
    risk_info = await risk_analyzer.get_model_info()

    return {
        "velocity_model": velocity_info,
        "risk_model": risk_info,
        "service_version": settings.MODEL_VERSION,
        "features": {
            "velocity_prediction": velocity_info["is_trained"],
            "risk_analysis

