from datetime import datetime, timedelta
from pathlib import Path

from portfolio_manager.services.auth_client import AuthClient
from portfolio_manager.services.kis.kis_token_manager import TokenManager
from portfolio_manager.services.kis.kis_token_store import (
    TokenData,
    MemoryTokenStore,
    FileTokenStore,
)


class FakeAuthClient(AuthClient):
    def __init__(self, token: TokenData) -> None:
        self._token = token
        self.calls = 0

    def request_access_token(self) -> TokenData:
        self.calls += 1
        return self._token


def test_token_manager_reuses_valid_token():
    now = datetime.now()
    stored = TokenData(token="cached", expires_at=now + timedelta(minutes=10))
    store = MemoryTokenStore()
    store.save(stored.token, stored.expires_at)

    auth = FakeAuthClient(TokenData(token="new", expires_at=now + timedelta(hours=1)))
    manager = TokenManager(store=store, auth_client=auth)

    token = manager.get_token()

    assert token == stored.token
    assert auth.calls == 0


def test_token_store_roundtrip():
    now = datetime.now()
    token = "access-token"
    expires_at = now + timedelta(hours=1)

    store = MemoryTokenStore()
    store.save(token, expires_at)

    loaded = store.load()

    assert loaded is not None
    assert loaded.token == token
    assert loaded.expires_at == expires_at


def test_file_token_store_roundtrip(tmp_path: Path):
    token_path = tmp_path / "token.json"
    store = FileTokenStore(token_path)

    expires_at = datetime.now() + timedelta(hours=1)
    store.save("file-token", expires_at)

    loaded = store.load()

    assert loaded is not None
    assert loaded.token == "file-token"
    assert loaded.expires_at == expires_at
