from datetime import datetime, timedelta
from pathlib import Path

from portfolio_manager.services.kis_token_store import FileTokenStore


def test_file_token_store_roundtrip(tmp_path: Path):
    token_path = tmp_path / "token.json"
    store = FileTokenStore(token_path)

    expires_at = datetime.now() + timedelta(hours=1)
    store.save("file-token", expires_at)

    loaded = store.load()

    assert loaded is not None
    assert loaded.token == "file-token"
    assert loaded.expires_at == expires_at
