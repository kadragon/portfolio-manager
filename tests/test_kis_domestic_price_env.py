from portfolio_manager.services.kis_domestic_price_client import KisDomesticPriceClient


def test_env_normalization_allows_whitespace_and_case():
    assert KisDomesticPriceClient._tr_id_for_env(" REAL ") == "FHKST01010100"
    assert KisDomesticPriceClient._tr_id_for_env("VPS ") == "FHKST01010100"
