from portfolio_manager.services.kis_domestic_price_client import KisDomesticPriceClient


def test_env_allows_real_prod_token():
    assert KisDomesticPriceClient._tr_id_for_env("real/prod") == "FHKST01010100"
