import asyncio
from typing import Optional
from datetime import datetime, timedelta
from loguru import logger

from app.config import get_settings
from app.services.data_service import DataService

settings = get_settings()


class TrainingService:
    """Service for managing model training and retraining."""
    
    def __init__(self):
        self.data_service = DataService()
        self.is_training = False
        self.last_training = None
        self.training_task = None
    
    async def schedule_retraining(self):
        """Schedule periodic retraining."""
        while True:
            try:
                await asyncio.sleep(settings.RETRAIN_INTERVAL_HOURS * 3600)  # Convert hours to seconds
                
                if not self.is_training:
                    logger.info("🔄 Starting scheduled retraining")
                    await self.retrain_models()
                else:
                    logger.info("⏭️ Skipping scheduled retraining (already in progress)")
                    
            except asyncio.CancelledError:
                logger.info("📅 Training scheduler cancelled")
                break
            except Exception as e:
                logger.error(f"❌ Scheduled retraining failed: {e}")
                # Continue the loop even if training fails
                await asyncio.sleep(3600)  # Wait 1 hour before retrying
    
    async def retrain_models(self, board_id: Optional[str] = None) -> dict:
        """Retrain all models with fresh data."""
        if self.is_training:
            logger.warning("⚠️ Training already in progress")
            return {"status": "already_training"}
        
        self.is_training = True
        training_results = {}
        
        try:
            logger.info(f"🎯 Starting model retraining for board: {board_id or 'all boards'}")
            
            # Get training data
            training_data = await self.data_service.get_training_data(board_id)
            
            logger.info(f"📊 Training data retrieved: {training_data['data_points']}")
            
            # Import models here to avoid circular imports
            from app.models.predictor import VelocityPredictor
            from app.models.risk_analyzer import RiskAnalyzer
            
            # Train velocity predictor
            try:
                velocity_predictor = VelocityPredictor()
                
                if training_data['velocity_data']:
                    velocity_metrics = await velocity_predictor.train(
                        velocity_data=training_data['velocity_data'],
                        board_id=board_id
                    )
                    training_results['velocity_model'] = {
                        'status': 'success',
                        'metrics': velocity_metrics,
                        'data_points': len(training_data['velocity_data'])
                    }
                    logger.info("✅ Velocity model retraining completed")
                else:
                    training_results['velocity_model'] = {
                        'status': 'skipped',
                        'reason': 'insufficient_data'
                    }
                    logger.warning("⚠️ Skipped velocity model training (insufficient data)")
                    
            except Exception as e:
                logger.error(f"❌ Velocity model retraining failed: {e}")
                training_results['velocity_model'] = {
                    'status': 'failed',
                    'error': str(e)
                }
            
            # Train risk analyzer
            try:
                risk_analyzer = RiskAnalyzer()
                
                if training_data['tasks_data']:
                    risk_metrics = await risk_analyzer.train(
                        tasks_data=training_data['tasks_data'],
                        board_id=board_id
                    )
                    training_results['risk_model'] = {
                        'status': 'success',
                        'metrics': risk_metrics,
                        'data_points': len(training_data['tasks_data'])
                    }
                    logger.info("✅ Risk model retraining completed")
                else:
                    training_results['risk_model'] = {
                        'status': 'skipped',
                        'reason': 'insufficient_data'
                    }
                    logger.warning("⚠️ Skipped risk model training (insufficient data)")
                    
            except Exception as e:
                logger.error(f"❌ Risk model retraining failed: {e}")
                training_results['risk_model'] = {
                    'status': 'failed',
                    'error': str(e)
                }
            
            # Update training metadata
            self.last_training = datetime.now()
            training_results['completed_at'] = self.last_training.isoformat()
            training_results['board_id'] = board_id
            training_results['training_data'] = training_data['data_points']
            
            logger.info(f"🎉 Model retraining completed: {training_results}")
            
            return training_results
            
        except Exception as e:
            logger.error(f"❌ Model retraining failed: {e}")
            training_results['error'] = str(e)
            training_results['status'] = 'failed'
            return training_results
            
        finally:
            self.is_training = False
    
    async def is_retraining_needed(self) -> bool:
        """Check if retraining is needed based on time elapsed."""
        if not self.last_training:
            return True
        
        time_since_training = datetime.now() - self.last_training
        return time_since_training > timedelta(hours=settings.RETRAIN_INTERVAL_HOURS)
    
    def get_training_status(self) -> dict:
        """Get current training status."""
        return {
            'is_training': self.is_training,
            'last_training': self.last_training.isoformat() if self.last_training else None,
            'retraining_needed': asyncio.run(self.is_retraining_needed()),
            'next_scheduled_training': None,  # Could be calculated
            'training_interval_hours': settings.RETRAIN_INTERVAL_HOURS,
        }
