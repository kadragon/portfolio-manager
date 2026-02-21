from portfolio_manager.web.app import create_app


def test_create_app_mounts_static_files():
    app = create_app()
    mounted_paths = {
        path
        for route in app.routes
        for path in [getattr(route, "path", None)]
        if isinstance(path, str)
    }
    assert "/static" in mounted_paths


def test_base_layout_contains_skip_link_and_live_regions(client):
    response = client.get("/groups")

    assert response.status_code == 200
    body = response.text

    assert 'class="skip-link" href="#main-content"' in body
    assert 'id="main-content"' in body
    assert 'id="app-live-region"' in body
    assert 'aria-live="polite"' in body
    assert 'id="app-request-status"' in body
    assert ">대시보드<" in body
    assert ">리밸런싱<" in body
