from portfolio_manager.services.kis_price_parser import parse_korea_price


def test_parse_korea_price_returns_common_model():
    payload = {
        "output": {
            "stck_prpr": "73500",
            "stck_code": "005930",
            "hts_kor_isnm": "삼성전자",
        }
    }

    price = parse_korea_price(payload)

    assert price.symbol == "005930"
    assert price.name == "삼성전자"
    assert price.price == 73500
    assert price.market == "KR"
