from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
import json


@dataclass(frozen=True)
class TokenData:
    token: str
    expires_at: datetime


class TokenStore(ABC):
    @abstractmethod
    def save(self, token: str, expires_at: datetime) -> None:
        """Save a token with expiration time."""

    @abstractmethod
    def load(self) -> TokenData | None:
        """Load a token if available."""


class MemoryTokenStore(TokenStore):
    def __init__(self) -> None:
        self._data: TokenData | None = None

    def save(self, token: str, expires_at: datetime) -> None:
        self._data = TokenData(token=token, expires_at=expires_at)

    def load(self) -> TokenData | None:
        return self._data


class FileTokenStore(TokenStore):
    def __init__(self, path: Path) -> None:
        self._path = path

    def save(self, token: str, expires_at: datetime) -> None:
        self._path.parent.mkdir(parents=True, exist_ok=True)
        payload = {
            "token": token,
            "expires_at": expires_at.isoformat(),
        }
        self._path.write_text(json.dumps(payload), encoding="utf-8")

    def load(self) -> TokenData | None:
        if not self._path.exists():
            return None
        payload = json.loads(self._path.read_text(encoding="utf-8"))
        return TokenData(
            token=payload["token"],
            expires_at=datetime.fromisoformat(payload["expires_at"]),
        )
