"""Tests for init_db default path resolution."""

import os
from unittest.mock import patch

from portfolio_manager.services.database import init_db


def test_init_db_uses_env_var_when_set(tmp_path):
    """init_db should use PORTFOLIO_DB_PATH env var when set."""
    db_file = str(tmp_path / "custom.db")
    with patch.dict(os.environ, {"PORTFOLIO_DB_PATH": db_file}):
        result = init_db()
    assert os.path.exists(db_file)
    result.close()


def test_init_db_default_uses_project_root():
    """init_db default path should be absolute, not relative to CWD."""
    from portfolio_manager.services.database import _default_db_path

    path = _default_db_path()
    assert os.path.isabs(path), f"Default DB path should be absolute, got: {path}"
