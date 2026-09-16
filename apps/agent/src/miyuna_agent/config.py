from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    agent_host: str = "127.0.0.1"
    agent_port: int = 8090
    go_api_url: str = "http://127.0.0.1:8080"
    internal_token: str = "dev-internal-token-change-me"
    ipc_timeout_seconds: float = 15.0
    # Tool execute/finalize can outlive regular IPC; must exceed API perToolTimeout (safety_analyzer = 30s).
    tool_execute_timeout_seconds: float = 45.0
    # LLM gateway calls can outlive regular IPC; must exceed API LLM_REQUEST_TIMEOUT (default 30s).
    llm_timeout_seconds: float = 120.0


settings = Settings()
