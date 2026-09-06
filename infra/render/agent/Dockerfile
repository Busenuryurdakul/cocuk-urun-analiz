# Miyuna Python Agent Orchestrator — internal service
FROM python:3.11-slim-bookworm
WORKDIR /app
COPY apps/agent/pyproject.toml apps/agent/README.md* ./
COPY apps/agent/src ./src
RUN pip install --no-cache-dir -e .
ENV AGENT_HOST=0.0.0.0
EXPOSE 8090
CMD ["sh", "-c", "uvicorn miyuna_agent.main:app --host 0.0.0.0 --port ${PORT:-8090}"]
