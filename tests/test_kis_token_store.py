from datetime import datetime, timedelta

from portfolio_manager.services.kis_token_store import TokenStore


def test_token_store_roundtrip():
    now = datetime.now()
    token = "access-token"
    expires_at = now + timedelta(hours=1)

    store = TokenStore()
    store.save(token, expires_at)

    loaded = store.load()

    assert loaded is not None
    assert loaded.token == token
    assert loaded.expires_at == expires_at
