from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    agent_host: str = "127.0.0.1"
    agent_port: int = 8090
    go_api_url: str = "http://127.0.0.1:8080"
    internal_token: str = "dev-internal-token-change-me"
    ipc_timeout_seconds: float = 15.0


settings = Settings()
