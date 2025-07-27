from fastapi import Request, HTTPException
from app.config import get_settings

settings = get_settings()


async def auth_middleware(request: Request, call_next):
    """Authentication middleware for API key validation."""
    
    # Skip auth for health and docs endpoints
    if request.url.path in ["/health", "/docs", "/redoc", "/openapi.json"]:
        response = await call_next(request)
        return response
    
    # Check for API key
    api_key = request.headers.get("X-API-Key") or request.query_params.get("api_key")
    
    if not api_key:
        raise HTTPException(status_code=401, detail="API key required")
    
    if api_key != settings.API_KEY:
        raise HTTPException(status_code=403, detail="Invalid API key")
    
    response = await call_next(request)
    return response
