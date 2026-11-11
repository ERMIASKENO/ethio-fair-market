# syntax=docker/dockerfile:1.7
# ---------------------------------------------------------------------------------------------
# ethio-fair-market / locations (Python, FastAPI)   build context: repository root
#   docker build -f services/locations/Dockerfile -t efm-locations .
# ---------------------------------------------------------------------------------------------
ARG PYTHON_VERSION=3.13

FROM python:${PYTHON_VERSION}-slim AS build
ENV PIP_DISABLE_PIP_VERSION_CHECK=1 PIP_NO_CACHE_DIR=1
RUN python -m venv /venv
COPY services/locations/requirements.txt services/locations/requirements-dev.txt /tmp/
RUN --mount=type=cache,target=/root/.cache/pip /venv/bin/pip install -r /tmp/requirements.txt
WORKDIR /app
COPY services/locations/app ./app
# compile once at build time: the container runs with a read-only filesystem
RUN /venv/bin/python -m compileall -q app /venv

FROM build AS test
RUN /venv/bin/pip install -r /tmp/requirements-dev.txt
COPY services/locations/tests ./tests
COPY services/locations/pyproject.toml ./
RUN /venv/bin/python -m pytest -q

FROM python:${PYTHON_VERSION}-slim AS runtime
ARG VERSION=dev
LABEL org.opencontainers.image.title="efm-locations" \
      org.opencontainers.image.description="ethio-fair-market locations service" \
      org.opencontainers.image.source="https://github.com/ermiaskeno/ethio-fair-market" \
      org.opencontainers.image.version="${VERSION}"
# security: newest OS patches, non-root user, no pip in the final image
RUN apt-get update && apt-get upgrade -y --no-install-recommends && rm -rf /var/lib/apt/lists/* \
 && useradd --uid 10001 --no-create-home --shell /usr/sbin/nologin app \
 && python -m pip uninstall -y pip setuptools wheel 2>/dev/null || true
COPY --from=build /venv /venv
WORKDIR /app
COPY --from=build /app/app ./app
ENV PATH=/venv/bin:$PATH PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 APP_VERSION=${VERSION}
USER 10001:10001
EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080", "--no-access-log", \
     "--proxy-headers", "--forwarded-allow-ips=*", "--timeout-graceful-shutdown", "20"]
