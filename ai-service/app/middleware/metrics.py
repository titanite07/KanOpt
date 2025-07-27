from fastapi import Request
import time
from prometheus_client import Counter, Histogram, generate_latest
from prometheus_client import CONTENT_TYPE_LATEST

# Metrics
REQUEST_COUNT = Counter('http_requests_total', 'Total HTTP requests', ['method', 'endpoint', 'status_code'])
REQUEST_DURATION = Histogram('http_request_duration_seconds', 'HTTP request duration', ['method', 'endpoint'])


async def metrics_middleware(request: Request, call_next):
    """Metrics collection middleware."""
    
    start_time = time.time()
    method = request.method
    endpoint = request.url.path
    
    # Skip metrics for metrics endpoint
    if endpoint == "/metrics":
        response = await call_next(request)
        return response
    
    try:
        response = await call_next(request)
        status_code = response.status_code
    except Exception as e:
        status_code = 500
        raise e
    finally:
        # Record metrics
        duration = time.time() - start_time
        REQUEST_COUNT.labels(method=method, endpoint=endpoint, status_code=status_code).inc()
        REQUEST_DURATION.labels(method=method, endpoint=endpoint).observe(duration)
    
    return response


async def get_metrics():
    """Get Prometheus metrics."""
    return generate_latest().decode('utf-8')
