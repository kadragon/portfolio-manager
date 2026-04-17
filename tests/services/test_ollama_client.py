"""Unit tests for OllamaClient."""

from __future__ import annotations

import httpx
import pytest

from portfolio_manager.services.llm.ollama_client import (
    OllamaClient,
    OllamaUnavailableError,
    ToolCall,
)


def _make_client(
    handler, *, model: str = "test-model", num_ctx: int | None = None
) -> OllamaClient:
    transport = httpx.MockTransport(handler)
    http = httpx.Client(transport=transport, base_url="http://localhost:11434")
    return OllamaClient(client=http, model=model, num_ctx=num_ctx)


def test_chat_returns_normalized_response_and_sends_expected_payload() -> None:
    captured: dict[str, object] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["path"] = request.url.path
        captured["method"] = request.method
        captured["body"] = request.content
        return httpx.Response(
            200,
            json={
                "model": "test-model",
                "done": True,
                "message": {
                    "role": "assistant",
                    "content": "  안녕하세요  ",
                },
            },
        )

    client = _make_client(handler)

    result = client.chat(
        [{"role": "user", "content": "hi"}],
        options={"temperature": 0.2},
    )

    assert captured["path"] == "/api/chat"
    assert captured["method"] == "POST"
    body = captured["body"]
    assert isinstance(body, bytes)
    assert b'"stream":false' in body
    assert b'"temperature":0.2' in body

    assert result.content == "안녕하세요"
    assert result.tool_calls == []
    assert result.model == "test-model"
    assert result.done is True


def test_chat_parses_tool_calls_with_dict_and_stringified_arguments() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            json={
                "model": "test-model",
                "done": True,
                "message": {
                    "role": "assistant",
                    "content": "",
                    "tool_calls": [
                        {
                            "function": {
                                "name": "get_group_summary",
                                "arguments": {"foo": "bar"},
                            }
                        },
                        {
                            "function": {
                                "name": "get_top_movers",
                                "arguments": '{"period": "1m", "n": 3}',
                            }
                        },
                        # malformed entries should be ignored
                        {"function": {"name": "", "arguments": {}}},
                        {"function": {"arguments": "not-json"}},
                    ],
                },
            },
        )

    client = _make_client(handler)
    result = client.chat([{"role": "user", "content": "q"}])

    assert result.tool_calls == [
        ToolCall(name="get_group_summary", arguments={"foo": "bar"}),
        ToolCall(name="get_top_movers", arguments={"period": "1m", "n": 3}),
    ]


def test_chat_merges_num_ctx_with_caller_options() -> None:
    captured: dict[str, object] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["body"] = request.content
        return httpx.Response(
            200,
            json={"model": "m", "done": True, "message": {"content": "ok"}},
        )

    client = _make_client(handler, num_ctx=4096)
    client.chat([{"role": "user", "content": "hi"}], options={"seed": 42})

    body = captured["body"]
    assert isinstance(body, bytes)
    assert b'"num_ctx":4096' in body
    assert b'"seed":42' in body


def test_chat_raises_unavailable_on_connection_error() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("refused")

    client = _make_client(handler)
    with pytest.raises(OllamaUnavailableError, match="연결 실패"):
        client.chat([{"role": "user", "content": "hi"}])


def test_chat_raises_unavailable_on_http_error_status() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(503, text="service unavailable")

    client = _make_client(handler)
    with pytest.raises(OllamaUnavailableError, match="status=503"):
        client.chat([{"role": "user", "content": "hi"}])


def test_chat_raises_unavailable_on_malformed_json() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(200, text="not-json")

    client = _make_client(handler)
    with pytest.raises(OllamaUnavailableError, match="파싱 실패"):
        client.chat([{"role": "user", "content": "hi"}])


def test_chat_raises_unavailable_when_response_is_not_object() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json=["nope"])

    client = _make_client(handler)
    with pytest.raises(OllamaUnavailableError, match="형식"):
        client.chat([{"role": "user", "content": "hi"}])


def test_chat_sends_tools_and_format_when_provided() -> None:
    captured: dict[str, object] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["body"] = request.content
        return httpx.Response(
            200,
            json={"model": "m", "done": True, "message": {"content": "{}"}},
        )

    client = _make_client(handler)
    client.chat(
        [{"role": "user", "content": "q"}],
        tools=[{"type": "function", "function": {"name": "tool_a"}}],
        format="json",
    )

    body = captured["body"]
    assert isinstance(body, bytes)
    assert b'"tools"' in body
    assert b'"format":"json"' in body
