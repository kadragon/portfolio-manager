"""KIS API business error types."""

from __future__ import annotations


class KisApiBusinessError(RuntimeError):
    """Raised when KIS returns a business error in a 200 response body."""

    def __init__(self, *, code: str, message: str) -> None:
        self.code = code.strip() or "UNKNOWN"
        self.message = message.strip() or "KIS API business error"
        super().__init__(f"{self.code}: {self.message}")
