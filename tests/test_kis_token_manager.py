from datetime import datetime, timedelta

from portfolio_manager.services.auth_client import AuthClient
from portfolio_manager.services.kis_token_manager import TokenManager
from portfolio_manager.services.kis_token_store import TokenData, MemoryTokenStore


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
