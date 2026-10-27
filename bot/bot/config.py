import os
from pathlib import Path
from dataclasses import dataclass

from dotenv import load_dotenv

ROOT_DIR = Path(__file__).resolve().parent.parent
load_dotenv(ROOT_DIR / ".env")


@dataclass(frozen=True)
class Config:
    bot_token: str
    api_url: str
    front_url: str
    bot_secret: str


def load_config() -> Config:
    token = os.getenv("BOT_TOKEN")
    if not token:
        raise RuntimeError("BOT_TOKEN не задан в .env")
    return Config(
        bot_token=token,
        api_url=os.getenv("API_URL", "http://localhost:8000/api/v1").rstrip("/"),
        front_url=os.getenv("FRONT_URL", "http://localhost:5173").rstrip("/"),
        bot_secret=os.getenv("BOT_SECRET", ""),
    )