from fastapi import FastAPI

from miyuna_agent import __version__

app = FastAPI(
    title="Miyuna Agent Orchestrator",
    description="Internal agent runtime — not exposed to public internet in production",
    version=__version__,
    docs_url="/docs" if False else None,  # no public docs in prod pattern
    redoc_url=None,
)


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok", "service": "miyuna-agent"}


@app.get("/ready")
def ready() -> dict[str, str]:
    return {"status": "ready", "service": "miyuna-agent"}
