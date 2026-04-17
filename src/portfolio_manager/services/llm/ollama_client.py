"""Sync Ollama chat client used by PortfolioInsightService."""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass, field
from typing import Any

import httpx

logger = logging.getLogger(__name__)


class OllamaUnavailableError(RuntimeError):
    """Raised when the Ollama server is unreachable or returns an invalid response."""


@dataclass
class ToolCall:
    """Tool call requested by the model."""

    name: str
    arguments: dict[str, Any]


@dataclass
class ChatResponse:
    """Normalized `/api/chat` response."""

    content: str
    tool_calls: list[ToolCall] = field(default_factory=list)
    model: str = ""
    done: bool = True
    raw: dict[str, Any] = field(default_factory=dict)


class OllamaClient:
    """Thin wrapper around Ollama's `/api/chat` endpoint.

    The caller supplies a pre-configured `httpx.Client` (base_url points at the
    Ollama server). Errors are funnelled through `OllamaUnavailableError` so
    routes can render a user-friendly fallback without leaking transport details.
    """

    def __init__(
        self,
        *,
        client: httpx.Client,
        model: str,
        timeout_sec: float = 60.0,
        num_ctx: int | None = None,
    ) -> None:
        self._client = client
        self._model = model
        self._timeout_sec = timeout_sec
        self._num_ctx = num_ctx

    @property
    def model(self) -> str:
        return self._model

    def chat(
        self,
        messages: list[dict[str, Any]],
        *,
        model: str | None = None,
        tools: list[dict[str, Any]] | None = None,
        format: str | dict[str, Any] | None = None,
        options: dict[str, Any] | None = None,
    ) -> ChatResponse:
        """Call `/api/chat` and return the normalized response."""

        payload: dict[str, Any] = {
            "model": model or self._model,
            "messages": messages,
            "stream": False,
        }
        if tools:
            payload["tools"] = tools
        if format is not None:
            payload["format"] = format

        merged_options: dict[str, Any] = {}
        if self._num_ctx is not None:
            merged_options["num_ctx"] = self._num_ctx
        if options:
            merged_options.update(options)
        if merged_options:
            payload["options"] = merged_options

        try:
            response = self._client.post(
                "/api/chat",
                json=payload,
                timeout=self._timeout_sec,
            )
        except httpx.HTTPError as exc:
            raise OllamaUnavailableError(f"Ollama 서버 연결 실패: {exc}") from exc

        if response.status_code >= 400:
            raise OllamaUnavailableError(
                f"Ollama 서버 오류: status={response.status_code}"
            )

        try:
            data = response.json()
        except ValueError as exc:
            raise OllamaUnavailableError(f"Ollama 응답 파싱 실패: {exc}") from exc

        if not isinstance(data, dict):
            raise OllamaUnavailableError("Ollama 응답 형식이 올바르지 않습니다.")

        message = data.get("message") or {}
        if not isinstance(message, dict):
            message = {}
        content = str(message.get("content") or "").strip()

        tool_calls = _parse_tool_calls(message.get("tool_calls"))

        return ChatResponse(
            content=content,
            tool_calls=tool_calls,
            model=str(data.get("model") or payload["model"]),
            done=bool(data.get("done", True)),
            raw=data,
        )


def _parse_tool_calls(raw_tool_calls: Any) -> list[ToolCall]:
    if not isinstance(raw_tool_calls, list):
        return []
    parsed: list[ToolCall] = []
    for entry in raw_tool_calls:
        if not isinstance(entry, dict):
            continue
        function = entry.get("function") or {}
        if not isinstance(function, dict):
            continue
        name = str(function.get("name") or "").strip()
        if not name:
            continue
        arguments = function.get("arguments")
        if isinstance(arguments, str):
            try:
                arguments = json.loads(arguments)
            except json.JSONDecodeError:
                arguments = {}
        if not isinstance(arguments, dict):
            arguments = {}
        parsed.append(ToolCall(name=name, arguments=arguments))
    return parsed
