from portfolio_manager.services.kis_price_parser import parse_us_price


def test_parse_us_price_returns_common_model():
    payload = {
        "output": {
            "last": "192.45",
            "symbol": "AAPL",
            "name": "Apple Inc",
        }
    }

    price = parse_us_price(payload)

    assert price.symbol == "AAPL"
    assert price.name == "Apple Inc"
    assert price.price == 192.45
    assert price.market == "US"
