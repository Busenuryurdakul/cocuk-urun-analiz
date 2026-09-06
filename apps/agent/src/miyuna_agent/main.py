from fastapi import Depends, FastAPI, Header, HTTPException
from pydantic import AliasChoices, BaseModel, Field

from miyuna_agent import __version__
from miyuna_agent.config import settings
from miyuna_agent.go_client import GoAgentClient
from miyuna_agent.llm_client import GoLLMClient
from miyuna_agent.run_manager import CancelRunPayload, RunManager, StartRunPayload

INTERNAL_TOKEN_HEADER = "X-Miyuna-Internal-Token"

app = FastAPI(
    title="Miyuna Agent Orchestrator",
    description="Internal agent runtime — not exposed to public internet in production",
    version=__version__,
    docs_url=None,
    redoc_url=None,
)

go_client = GoAgentClient(
    base_url=settings.go_api_url,
    token=settings.internal_token,
    timeout_seconds=settings.ipc_timeout_seconds,
)
llm_client = GoLLMClient(
    base_url=settings.go_api_url,
    token=settings.internal_token,
    timeout_seconds=settings.ipc_timeout_seconds,
)
run_manager = RunManager(go_client=go_client, llm_client=llm_client)


def require_internal_token(
    x_miyuna_internal_token: str | None = Header(default=None, alias=INTERNAL_TOKEN_HEADER),
) -> None:
    if not settings.internal_token or x_miyuna_internal_token != settings.internal_token:
        raise HTTPException(status_code=401, detail="unauthorized")


class StartRunRequest(BaseModel):
    organization_id: str = Field(validation_alias=AliasChoices("organizationId", "organization_id"))
    analysis_run_id: str = Field(validation_alias=AliasChoices("analysisRunId", "analysis_run_id"))
    product_id: str = Field(validation_alias=AliasChoices("productId", "product_id"))
    trace_id: str = Field(validation_alias=AliasChoices("traceId", "trace_id"))
    actor_user_id: str = Field(validation_alias=AliasChoices("actorUserId", "actor_user_id"))
    capabilities: dict = Field(default_factory=dict)

    model_config = {"populate_by_name": True}


class CancelRunRequest(BaseModel):
    organization_id: str = Field(validation_alias=AliasChoices("organizationId", "organization_id"))
    analysis_run_id: str = Field(validation_alias=AliasChoices("analysisRunId", "analysis_run_id"))
    trace_id: str = Field(validation_alias=AliasChoices("traceId", "trace_id"))

    model_config = {"populate_by_name": True}


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok", "service": "miyuna-agent"}


@app.get("/ready")
def ready() -> dict[str, str]:
    return {"status": "ready", "service": "miyuna-agent"}


@app.post("/internal/v1/runs/start", dependencies=[Depends(require_internal_token)])
def start_analysis_run(req: StartRunRequest) -> dict[str, str]:
    run_manager.start_run(
        StartRunPayload(
            organization_id=req.organization_id,
            analysis_run_id=req.analysis_run_id,
            product_id=req.product_id,
            trace_id=req.trace_id,
            actor_user_id=req.actor_user_id,
            capabilities=req.capabilities,
        )
    )
    return {"status": "accepted"}


@app.post("/internal/v1/runs/cancel", dependencies=[Depends(require_internal_token)])
def cancel_analysis_run(req: CancelRunRequest) -> dict[str, str]:
    run_manager.cancel_run(
        CancelRunPayload(
            organization_id=req.organization_id,
            analysis_run_id=req.analysis_run_id,
            trace_id=req.trace_id,
        )
    )
    return {"status": "accepted"}
