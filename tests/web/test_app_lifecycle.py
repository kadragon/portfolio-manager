import sys
from decimal import Decimal
from types import SimpleNamespace

from fastapi import FastAPI
from fastapi.testclient import TestClient

import portfolio_manager.web.app as web_app


def test_format_krw_filter(app: FastAPI):
    filters = app.state.templates.env.filters
    assert filters["format_krw"](None) == "-"
    assert filters["format_krw"]("1234") == "₩1,234"


def test_format_usd_filter(app: FastAPI):
    filters = app.state.templates.env.filters
    assert filters["format_usd"](None) == "-"
    assert filters["format_usd"]("12.3") == "$12.30"


def test_format_percent_filter(app: FastAPI):
    filters = app.state.templates.env.filters
    assert filters["format_percent"](None) == "-"
    assert filters["format_percent"]("1.25") == "1.3%"


def test_format_signed_percent_filter(app: FastAPI):
    filters = app.state.templates.env.filters
    assert filters["format_signed_percent"](None) == "-"
    assert filters["format_signed_percent"]("1.25") == "+1.3%"
    assert filters["format_signed_percent"]("-1.25") == "-1.3%"


def test_abs_filter(app: FastAPI):
    filters = app.state.templates.env.filters
    assert filters["abs"]("-3.5") == Decimal("3.5")
    assert filters["abs"](None) is None


def test_lifespan_loads_env_sets_up_container_and_closes_on_shutdown(monkeypatch):
    class DummyContainer:
        # Class-level list is safe: DummyContainer is redefined on each test
        # invocation because it is declared inside the test function body.
        instances: list = []

        def __init__(self):
            self.setup_called = False
            self.close_called = False
            DummyContainer.instances.append(self)

        def setup(self):
            self.setup_called = True

        def close(self):
            self.close_called = True

    calls = {"load_dotenv": 0}

    def fake_load_dotenv():
        calls["load_dotenv"] += 1

    monkeypatch.setattr(web_app, "ServiceContainer", DummyContainer)
    monkeypatch.setattr(web_app, "load_dotenv", fake_load_dotenv)

    app = web_app.create_app()

    with TestClient(app):
        container = app.state.container
        assert isinstance(container, DummyContainer)
        assert container.setup_called is True
        assert calls["load_dotenv"] == 1

    assert DummyContainer.instances[0].close_called is True


def test_run_invokes_uvicorn_with_expected_arguments(monkeypatch):
    captured: dict[str, object] = {}

    def fake_run(*args, **kwargs):
        captured["args"] = args
        captured["kwargs"] = kwargs

    monkeypatch.setitem(sys.modules, "uvicorn", SimpleNamespace(run=fake_run))

    web_app.run()

    assert captured["args"] == ("portfolio_manager.web.app:app",)
    assert captured["kwargs"] == {
        "host": "127.0.0.1",
        "port": 8000,
        "reload": True,
    }
