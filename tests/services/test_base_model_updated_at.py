"""Tests for BaseModel.save() auto-updating updated_at."""

from datetime import datetime, timezone

from portfolio_manager.services.database import GroupModel, init_db


def test_save_auto_updates_updated_at(tmp_path):
    """When saving an existing model, updated_at should be refreshed automatically."""
    db_path = str(tmp_path / "test.db")
    db = init_db(db_path)

    try:
        row = GroupModel.create(name="test", target_percentage=10.0)

        # Force a past timestamp to detect change
        past = datetime(2020, 1, 1, tzinfo=timezone.utc)
        GroupModel.update(updated_at=past).where(GroupModel.id == row.id).execute()
        row = GroupModel.get_by_id(row.id)
        assert row.updated_at == past

        # save() should auto-update updated_at
        row.name = "updated"
        row.save()

        row = GroupModel.get_by_id(row.id)
        assert row.updated_at > past
    finally:
        db.close()
