import sys
from decimal import Decimal
from pathlib import Path
from types import SimpleNamespace

from fastapi.templating import Jinja2Templates
from fastapi.testclient import TestClient

import portfolio_manager.web.app as web_app


def _templates() -> Jinja2Templates:
    templates_dir = (
        Path(__file__).resolve().parents[2]
        / "src"
        / "portfolio_manager"
        / "web"
        / "templates"
    )
    return Jinja2Templates(directory=str(templates_dir))


def test_add_filters_handles_none_rounding_sign_and_abs():
    templates = _templates()
    web_app._add_filters(templates)
    filters = templates.env.filters

    assert filters["format_krw"](None) == "-"
    assert filters["format_krw"]("1234") == "₩1,234"
    assert filters["format_usd"](None) == "-"
    assert filters["format_usd"]("12.3") == "$12.30"
    assert filters["format_percent"](None) == "-"
    assert filters["format_percent"]("1.25") == "1.3%"
    assert filters["format_signed_percent"](None) == "-"
    assert filters["format_signed_percent"]("1.25") == "+1.3%"
    assert filters["format_signed_percent"]("-1.25") == "-1.3%"
    assert filters["abs"]("-3.5") == Decimal("3.5")
    assert filters["abs"](None) is None


def test_lifespan_loads_env_sets_up_container_and_closes_on_shutdown(monkeypatch):
    class DummyContainer:
        instances = []

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
