from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    agent_host: str = "127.0.0.1"
    agent_port: int = 8090


settings = Settings()
