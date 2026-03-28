"""Shared fixtures for repository tests using in-memory SQLite."""

import pytest

from portfolio_manager.services.database import db, ALL_MODELS


@pytest.fixture(autouse=True)
def setup_test_db():
    """Initialize in-memory SQLite for each test."""
    db.init(":memory:", pragmas={"foreign_keys": 1})
    db.create_tables(ALL_MODELS)
    yield
    db.drop_tables(ALL_MODELS)
    db.close()
