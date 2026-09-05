from fastapi.testclient import TestClient

from miyuna_agent.config import settings
from miyuna_agent.main import app


def test_python_start_run_requires_internal_token() -> None:
    client = TestClient(app)
    resp = client.post(
        "/internal/v1/runs/start",
        json={
            "organizationId": "org",
            "analysisRunId": "run",
            "productId": "prod",
            "traceId": "trace",
            "actorUserId": "user",
            "capabilities": {},
        },
    )
    assert resp.status_code == 401


def test_python_start_run_accepts_with_token(monkeypatch) -> None:
    monkeypatch.setattr("miyuna_agent.main.run_manager.start_run", lambda payload: None)
    client = TestClient(app)
    resp = client.post(
        "/internal/v1/runs/start",
        json={
            "organizationId": "org",
            "analysisRunId": "run",
            "productId": "prod",
            "traceId": "trace",
            "actorUserId": "user",
            "capabilities": {},
        },
        headers={"X-Miyuna-Internal-Token": settings.internal_token},
    )
    assert resp.status_code == 200
    assert resp.json()["status"] == "accepted"
