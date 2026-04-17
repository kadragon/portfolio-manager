"""Time utilities — KST is the project-wide timezone."""

from __future__ import annotations

from datetime import datetime
from zoneinfo import ZoneInfo

KST = ZoneInfo("Asia/Seoul")


def now_kst() -> datetime:
    """Return the current time in KST (Asia/Seoul)."""
    return datetime.now(KST)
