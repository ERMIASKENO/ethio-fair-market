"""locations: Ethiopian regions, cities and sub-cities for forms, filters and validation."""

from __future__ import annotations

import json
import logging
import os
import sys
import time
import uuid
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, Query, Request, Response
from fastapi.responses import JSONResponse
from prometheus_client import CONTENT_TYPE_LATEST, Counter, Histogram, generate_latest

from .data import BY_CODE, REGIONS, City, Region, find_city

SERVICE = "locations"
VERSION = os.getenv("APP_VERSION", "dev")


class JsonFormatter(logging.Formatter):
    """One JSON object per line - same shape as the Go and Node services."""

    def format(self, record: logging.LogRecord) -> str:
        entry = {"time": self.formatTime(record, "%Y-%m-%dT%H:%M:%S%z"), "level": record.levelname, "msg": record.getMessage(),
                 "service": SERVICE, "version": VERSION}
        entry.update(getattr(record, "extra_fields", {}))
        return json.dumps(entry, ensure_ascii=False)


handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(JsonFormatter())
log = logging.getLogger(SERVICE)
log.addHandler(handler)
log.setLevel(os.getenv("LOG_LEVEL", "INFO").upper())
log.propagate = False

REQUESTS = Counter("http_requests_total", "HTTP requests handled.", ["route", "code"])
LATENCY = Histogram("http_request_duration_seconds", "HTTP request latency.", ["route"],
                    buckets=(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5))

state = {"shutting_down": False}


@asynccontextmanager
async def lifespan(_: FastAPI):
    log.info("started", extra={"extra_fields": {"regions": len(REGIONS)}})
    yield
    # Kubernetes removes the pod from the Service endpoints on termination (preStop sleep covers the delay);
    # uvicorn then stops accepting connections and lets open requests finish.
    state["shutting_down"] = True
    log.info("shutdown complete")


app = FastAPI(title="ethio-fair-market locations", version=VERSION, docs_url="/v1/locations/docs",
              openapi_url="/v1/locations/openapi.json", redoc_url=None, lifespan=lifespan)

SKIP = {"/healthz", "/readyz", "/metrics"}


@app.middleware("http")
async def observability(request: Request, call_next):
    if request.url.path in SKIP:
        return await call_next(request)
    start = time.perf_counter()
    request_id = request.headers.get("x-request-id") or uuid.uuid4().hex[:24]
    response = await call_next(request)
    route = request.scope.get("route")
    route_label = f"{request.method} {route.path}" if route is not None else "unmatched"
    seconds = time.perf_counter() - start
    REQUESTS.labels(route_label, str(response.status_code)).inc()
    LATENCY.labels(route_label).observe(seconds)
    response.headers["X-Request-Id"] = request_id
    response.headers["X-Content-Type-Options"] = "nosniff"
    log.info("request", extra={"extra_fields": {"request_id": request_id, "method": request.method, "path": request.url.path,
                                               "route": route_label, "status": response.status_code,
                                               "duration_ms": round(seconds * 1000)}})
    return response


def city_view(c: City, with_subcities: bool = True) -> dict:
    out = {"slug": c.slug, "name": {"en": c.name, "am": c.name_am}}
    if with_subcities:
        out["subcities"] = [{"name": {"en": en, "am": am}} for en, am in c.subcities]
    return out


def region_view(r: Region, with_cities: bool) -> dict:
    out = {"code": r.code, "name": {"en": r.name, "am": r.name_am}, "kind": r.kind}
    if with_cities:
        out["cities"] = [city_view(c) for c in r.cities]
    return out


def not_found(message: str) -> HTTPException:
    return HTTPException(status_code=404, detail={"code": "not_found", "message": message})


@app.exception_handler(HTTPException)
async def http_error(_: Request, exc: HTTPException):
    detail = exc.detail if isinstance(exc.detail, dict) else {"code": "error", "message": str(exc.detail)}
    return JSONResponse(status_code=exc.status_code, content={"error": detail})


CACHE = {"Cache-Control": "public, max-age=3600"}


@app.get("/v1/locations/regions")
def list_regions(response: Response, include_cities: bool = False):
    response.headers.update(CACHE)
    return {"items": [region_view(r, include_cities) for r in REGIONS]}


@app.get("/v1/locations/regions/{code}")
def get_region(code: str, response: Response):
    region = BY_CODE.get(code.upper())
    if region is None:
        raise not_found(f"unknown region {code}")
    response.headers.update(CACHE)
    return region_view(region, with_cities=True)


@app.get("/v1/locations/regions/{code}/cities/{city}")
def get_city(code: str, city: str, response: Response):
    region = BY_CODE.get(code.upper())
    found = find_city(region, city) if region else None
    if found is None:
        raise not_found(f"unknown city {city} in region {code}")
    response.headers.update(CACHE)
    return {"region": region_view(region, with_cities=False), **city_view(found)}


@app.get("/v1/locations/search")
def search(q: str = Query(min_length=1, max_length=50), limit: int = Query(10, ge=1, le=50)):
    """Autocomplete across cities and sub-cities in English and Amharic."""
    term = q.strip().lower()
    hits = []
    for r in REGIONS:
        for c in r.cities:
            if term in c.name.lower() or term in c.name_am:
                hits.append({"type": "city", "region": r.code, "city": c.name, "name": {"en": c.name, "am": c.name_am}})
            for en, am in c.subcities:
                if term in en.lower() or term in am:
                    hits.append({"type": "subcity", "region": r.code, "city": c.name, "subcity": en, "name": {"en": en, "am": am}})
    return {"items": hits[:limit]}


@app.get("/healthz")
def healthz():
    return {"status": "ok"}


@app.get("/readyz")
def readyz():
    if state["shutting_down"]:
        return JSONResponse(status_code=503, content={"status": "shutting down"})
    return {"status": "ready", "regions": len(REGIONS)}


@app.get("/metrics")
def metrics():
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)
