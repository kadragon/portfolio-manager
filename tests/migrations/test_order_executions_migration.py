"""Tests for order_executions migration."""

from pathlib import Path

MIGRATION_PATH = Path("supabase/migrations/20260209000000_create_order_executions.sql")


def test_order_executions_migration_exists_and_allows_duplicate_records():
    """Migration should create order_executions without unique constraints."""

    assert MIGRATION_PATH.exists(), f"Migration file not found: {MIGRATION_PATH}"

    sql = MIGRATION_PATH.read_text().lower()

    assert "create table" in sql
    assert "order_executions" in sql

    # Required columns
    for column in ["ticker", "side", "quantity", "currency", "status", "message"]:
        assert column in sql, f"Missing column: {column}"

    assert "exchange" in sql
    assert "raw_response" in sql

    # No unique constraint on business columns (allow multiple history entries)
    # Only the primary key should be unique
    lines_without_pk = [line for line in sql.splitlines() if "primary key" not in line]
    remaining = "\n".join(lines_without_pk)
    assert "unique" not in remaining, (
        "order_executions should not have unique constraints"
    )
