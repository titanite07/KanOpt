import asyncio
import asyncpg
import httpx
from typing import List, Dict, Any, Optional
from datetime import datetime, timedelta
from loguru import logger

from app.config import get_settings

settings = get_settings()


class DataService:
    """Service for data retrieval and processing."""
    
    def __init__(self):
        self.db_pool = None
        self.http_client = None
    
    async def initialize(self):
        """Initialize database connection and HTTP client."""
        try:
            # Initialize database pool
            self.db_pool = await asyncpg.create_pool(
                settings.DATABASE_URL,
                min_size=2,
                max_size=10,
                command_timeout=60
            )
            
            # Initialize HTTP client
            self.http_client = httpx.AsyncClient(
                base_url=settings.KANBAN_API_URL,
                timeout=30.0
            )
            
            logger.info("✅ Data service initialized")
            
        except Exception as e:
            logger.error(f"❌ Failed to initialize data service: {e}")
            raise
    
    async def close(self):
        """Close connections."""
        if self.db_pool:
            await self.db_pool.close()
        
        if self.http_client:
            await self.http_client.aclose()
    
    async def get_velocity_history(self, board_id: str, weeks: int = 12) -> List[Dict[str, Any]]:
        """Get velocity history for a board."""
        try:
            if not self.db_pool:
                await self.initialize()
            
            query = """
                SELECT 
                    sprint_week,
                    velocity,
                    completed,
                    total_points,
                    cycle_time,
                    throughput,
                    created_at
                FROM velocity_metrics 
                WHERE board_id = $1 
                ORDER BY sprint_week DESC 
                LIMIT $2
            """
            
            async with self.db_pool.acquire() as conn:
                rows = await conn.fetch(query, board_id, weeks)
                
                return [
                    {
                        'sprint_week': row['sprint_week'],
                        'velocity': float(row['velocity']),
                        'completed': row['completed'],
                        'total_story_points': row['total_points'],
                        'cycle_time': float(row['cycle_time']),
                        'throughput': row['throughput'],
                        'created_at': row['created_at'].isoformat(),
                    }
                    for row in rows
                ]
                
        except Exception as e:
            logger.error(f"❌ Failed to get velocity history: {e}")
            # Fallback to API if database fails
            return await self.get_velocity_history_from_api(board_id)
    
    async def get_velocity_history_from_api(self, board_id: str) -> List[Dict[str, Any]]:
        """Get velocity history from main API as fallback."""
        try:
            if not self.http_client:
                await self.initialize()
            
            response = await self.http_client.get(f"/api/analytics/board/{board_id}/velocity")
            response.raise_for_status()
            
            data = response.json()
            return data.get('weeklyMetrics', [])
            
        except Exception as e:
            logger.error(f"❌ Failed to get velocity from API: {e}")
            return []
    
    async def get_board_tasks(self, board_id: str, include_completed: bool = False) -> List[Dict[str, Any]]:
        """Get tasks for a board."""
        try:
            if not self.db_pool:
                await self.initialize()
            
            query = """
                SELECT 
                    id,
                    title,
                    description,
                    priority,
                    story_points,
                    assignee_id,
                    due_date,
                    created_at,
                    updated_at,
                    completed_at
                FROM tasks 
                WHERE board_id = $1
            """
            
            if not include_completed:
                query += " AND completed_at IS NULL"
            
            query += " ORDER BY created_at DESC"
            
            async with self.db_pool.acquire() as conn:
                rows = await conn.fetch(query, board_id)
                
                return [
                    {
                        'id': str(row['id']),
                        'title': row['title'],
                        'description': row['description'] or '',
                        'priority': row['priority'],
                        'storyPoints': row['story_points'],
                        'assigneeId': str(row['assignee_id']) if row['assignee_id'] else None,
                        'dueDate': row['due_date'].isoformat() if row['due_date'] else None,
                        'createdAt': row['created_at'].isoformat(),
                        'updatedAt': row['updated_at'].isoformat(),
                        'completedAt': row['completed_at'].isoformat() if row['completed_at'] else None,
                    }
                    for row in rows
                ]
                
        except Exception as e:
            logger.error(f"❌ Failed to get board tasks: {e}")
            # Fallback to API
            return await self.get_board_tasks_from_api(board_id)
    
    async def get_board_tasks_from_api(self, board_id: str) -> List[Dict[str, Any]]:
        """Get tasks from main API as fallback."""
        try:
            if not self.http_client:
                await self.initialize()
            
            response = await self.http_client.get(f"/api/tasks?boardId={board_id}")
            response.raise_for_status()
            
            return response.json()
            
        except Exception as e:
            logger.error(f"❌ Failed to get tasks from API: {e}")
            return []
    
    async def get_all_boards(self) -> List[Dict[str, Any]]:
        """Get all board IDs for training."""
        try:
            if not self.db_pool:
                await self.initialize()
            
            query = "SELECT id, name FROM boards ORDER BY created_at DESC"
            
            async with self.db_pool.acquire() as conn:
                rows = await conn.fetch(query)
                
                return [
                    {
                        'id': str(row['id']),
                        'name': row['name'],
                    }
                    for row in rows
                ]
                
        except Exception as e:
            logger.error(f"❌ Failed to get boards: {e}")
            return []
    
    async def get_training_data(self, board_id: Optional[str] = None) -> Dict[str, Any]:
        """Get comprehensive training data."""
        try:
            boards = [{'id': board_id}] if board_id else await self.get_all_boards()
            
            all_velocity_data = []
            all_tasks_data = []
            
            for board in boards:
                board_id = board['id']
                
                # Get velocity data
                velocity_data = await self.get_velocity_history(board_id, weeks=26)  # 6 months
                all_velocity_data.extend(velocity_data)
                
                # Get tasks data
                tasks_data = await self.get_board_tasks(board_id, include_completed=True)
                all_tasks_data.extend(tasks_data)
            
            return {
                'velocity_data': all_velocity_data,
                'tasks_data': all_tasks_data,
                'boards_count': len(boards),
                'data_points': {
                    'velocity': len(all_velocity_data),
                    'tasks': len(all_tasks_data),
                }
            }
            
        except Exception as e:
            logger.error(f"❌ Failed to get training data: {e}")
            raise
