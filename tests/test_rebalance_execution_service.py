"""Tests for RebalanceExecutionService."""

from decimal import Decimal
from unittest.mock import MagicMock

from portfolio_manager.models.rebalance import (
    RebalanceAction,
    RebalanceRecommendation,
)
from portfolio_manager.services.rebalance_execution_service import (
    RebalanceExecutionService,
)


def test_creates_order_intents_from_recommendations():
    """RebalanceExecutionService가 리밸런싱 추천에서 OrderIntent 목록을 생성한다."""
    recommendations = [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=1,
            currency="KRW",
            quantity=Decimal("7.3"),
            stock_name="삼성전자",
            group_name="Growth",
        ),
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("300"),
            priority=2,
            currency="USD",
            quantity=Decimal("2.0"),
            stock_name="Apple Inc",
            group_name="Tech",
        ),
    ]

    service = RebalanceExecutionService()
    result = service.create_order_intents(recommendations)

    assert len(result.intents) == 2

    # Sells are ordered before buys
    sell_intent = result.intents[0]
    assert sell_intent.ticker == "AAPL"
    assert sell_intent.side == "sell"
    assert sell_intent.quantity == 2
    assert sell_intent.currency == "USD"

    buy_intent = result.intents[1]
    assert buy_intent.ticker == "005930"
    assert buy_intent.side == "buy"
    assert buy_intent.quantity == 7
    assert buy_intent.currency == "KRW"


def test_fractional_quantity_is_floored_to_integer():
    """수량이 소수면 주문 전 정수 수량으로 정규화한다(기본: floor)."""
    recommendations = [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=1,
            currency="KRW",
            quantity=Decimal("7.9"),
        ),
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("300"),
            priority=2,
            currency="USD",
            quantity=Decimal("3.1"),
        ),
    ]

    service = RebalanceExecutionService()
    result = service.create_order_intents(recommendations)

    # Sells sorted before buys: AAPL(sell) then 005930(buy)
    assert result.intents[0].quantity == 3  # 3.1 -> floor -> 3
    assert result.intents[1].quantity == 7  # 7.9 -> floor -> 7
    assert all(isinstance(i.quantity, int) for i in result.intents)


def test_zero_quantity_after_floor_is_skipped():
    """정규화 결과 0주가 되면 해당 주문을 skipped로 분류한다."""
    recommendations = [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=1,
            currency="KRW",
            quantity=Decimal("5.0"),
        ),
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.BUY,
            amount=Decimal("50"),
            priority=2,
            currency="USD",
            quantity=Decimal("0.3"),  # floor -> 0 -> skipped
        ),
        RebalanceRecommendation(
            ticker="TSLA",
            action=RebalanceAction.SELL,
            amount=Decimal("10"),
            priority=3,
            currency="USD",
            quantity=None,  # None -> 0 -> skipped
        ),
    ]

    service = RebalanceExecutionService()
    result = service.create_order_intents(recommendations)

    assert len(result.intents) == 1
    assert result.intents[0].ticker == "005930"

    assert len(result.skipped) == 2
    assert result.skipped[0].ticker == "AAPL"
    assert result.skipped[1].ticker == "TSLA"


def test_sell_intents_ordered_before_buy_intents():
    """Sell intent가 Buy intent보다 먼저 실행 순서에 배치된다."""
    recommendations = [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=1,
            currency="KRW",
            quantity=Decimal("10"),
        ),
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("300"),
            priority=2,
            currency="USD",
            quantity=Decimal("3"),
        ),
        RebalanceRecommendation(
            ticker="MSFT",
            action=RebalanceAction.BUY,
            amount=Decimal("200"),
            priority=3,
            currency="USD",
            quantity=Decimal("1"),
        ),
        RebalanceRecommendation(
            ticker="000660",
            action=RebalanceAction.SELL,
            amount=Decimal("100000"),
            priority=4,
            currency="KRW",
            quantity=Decimal("2"),
        ),
    ]

    service = RebalanceExecutionService()
    result = service.create_order_intents(recommendations)

    assert len(result.intents) == 4
    # All sells come before all buys
    assert result.intents[0].side == "sell"
    assert result.intents[1].side == "sell"
    assert result.intents[2].side == "buy"
    assert result.intents[3].side == "buy"


def test_overseas_intent_uses_stock_exchange_or_default():
    """해외 주문 intent는 stock.exchange 우선, 없으면 기본 NASD를 사용한다."""
    recommendations = [
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("300"),
            priority=1,
            currency="USD",
            quantity=Decimal("2"),
        ),
        RebalanceRecommendation(
            ticker="MSFT",
            action=RebalanceAction.BUY,
            amount=Decimal("200"),
            priority=2,
            currency="USD",
            quantity=Decimal("1"),
        ),
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=3,
            currency="KRW",
            quantity=Decimal("10"),
        ),
    ]

    # AAPL has cached exchange NYSE, MSFT has no cached exchange
    exchange_map = {"AAPL": "NYSE", "MSFT": None}

    service = RebalanceExecutionService()
    result = service.create_order_intents(recommendations, exchange_map=exchange_map)

    # AAPL uses cached exchange NYSE
    aapl = next(i for i in result.intents if i.ticker == "AAPL")
    assert aapl.exchange == "NYSE"

    # MSFT has no cached exchange -> default NASD
    msft = next(i for i in result.intents if i.ticker == "MSFT")
    assert msft.exchange == "NASD"

    # Domestic ticker -> no exchange needed (None)
    samsung = next(i for i in result.intents if i.ticker == "005930")
    assert samsung.exchange is None


def test_dry_run_returns_intents_without_api_calls():
    """execute_rebalance_orders(dry_run=True)는 API 호출 없이 intent 요약만 반환한다."""
    recommendations = [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=1,
            currency="KRW",
            quantity=Decimal("7"),
        ),
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("300"),
            priority=2,
            currency="USD",
            quantity=Decimal("2"),
        ),
    ]

    mock_order_client = MagicMock()
    service = RebalanceExecutionService(order_client=mock_order_client)
    result = service.execute_rebalance_orders(recommendations, dry_run=True)

    # No API calls made
    mock_order_client.place_order.assert_not_called()

    # Returns intents as summary
    assert len(result.intents) == 2
    assert result.intents[0].side == "sell"  # sell first
    assert result.intents[1].side == "buy"


def test_execute_calls_order_api_in_intent_order():
    """execute_rebalance_orders(dry_run=False)는 intent 순서대로 주문 API를 호출한다."""
    recommendations = [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=1,
            currency="KRW",
            quantity=Decimal("7"),
        ),
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("300"),
            priority=2,
            currency="USD",
            quantity=Decimal("2"),
        ),
    ]

    mock_order_client = MagicMock()
    mock_order_client.place_order.return_value = {
        "rt_cd": "0",
        "msg_cd": "APBK0013",
        "msg1": "주문 전송 완료",
    }

    service = RebalanceExecutionService(order_client=mock_order_client)
    result = service.execute_rebalance_orders(recommendations, dry_run=False)

    # API called once per intent (sell first, then buy)
    assert mock_order_client.place_order.call_count == 2

    first_call = mock_order_client.place_order.call_args_list[0]
    assert first_call.kwargs["ticker"] == "AAPL"
    assert first_call.kwargs["side"] == "sell"

    second_call = mock_order_client.place_order.call_args_list[1]
    assert second_call.kwargs["ticker"] == "005930"
    assert second_call.kwargs["side"] == "buy"

    # All executions recorded
    assert result.executions is not None
    assert len(result.executions) == 2
    assert result.executions[0].status == "success"
    assert result.executions[1].status == "success"


def test_single_failure_does_not_stop_remaining_orders():
    """주문 API 실패 1건이 있어도 나머지 주문은 계속 실행하고 개별 실패를 수집한다."""
    recommendations = [
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("300"),
            priority=1,
            currency="USD",
            quantity=Decimal("2"),
        ),
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=2,
            currency="KRW",
            quantity=Decimal("7"),
        ),
        RebalanceRecommendation(
            ticker="000660",
            action=RebalanceAction.BUY,
            amount=Decimal("200000"),
            priority=3,
            currency="KRW",
            quantity=Decimal("3"),
        ),
    ]

    fail_response = {
        "rt_cd": "1",
        "msg_cd": "APBK0919",
        "msg1": "주문가능수량 부족",
    }
    success_response = {
        "rt_cd": "0",
        "msg_cd": "APBK0013",
        "msg1": "주문 전송 완료",
    }

    mock_order_client = MagicMock()
    # First call (AAPL sell) succeeds, second (005930 buy) fails, third (000660 buy) succeeds
    mock_order_client.place_order.side_effect = [
        success_response,
        fail_response,
        success_response,
    ]

    service = RebalanceExecutionService(order_client=mock_order_client)
    result = service.execute_rebalance_orders(recommendations, dry_run=False)

    # All 3 orders were attempted despite the middle failure
    assert mock_order_client.place_order.call_count == 3

    # Individual results collected correctly
    assert result.executions is not None
    assert len(result.executions) == 3
    assert result.executions[0].status == "success"
    assert result.executions[1].status == "failed"
    assert result.executions[1].message == "주문가능수량 부족"
    assert result.executions[2].status == "success"


def test_execution_results_are_persisted():
    """실행 결과가 order_executions에 저장된다(성공/실패/스킵 모두 기록)."""
    recommendations = [
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("300"),
            priority=1,
            currency="USD",
            quantity=Decimal("2"),
        ),
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=2,
            currency="KRW",
            quantity=Decimal("7"),
        ),
        RebalanceRecommendation(
            ticker="TSLA",
            action=RebalanceAction.BUY,
            amount=Decimal("10"),
            priority=3,
            currency="USD",
            quantity=Decimal("0.3"),  # floor -> 0 -> skipped
        ),
    ]

    mock_order_client = MagicMock()
    mock_order_client.place_order.return_value = {
        "rt_cd": "0",
        "msg_cd": "APBK0013",
        "msg1": "주문 전송 완료",
    }
    mock_repo = MagicMock()

    service = RebalanceExecutionService(
        order_client=mock_order_client,
        execution_repository=mock_repo,
    )
    service.execute_rebalance_orders(recommendations, dry_run=False)

    # Repository create called for each execution + each skipped
    assert mock_repo.create.call_count == 3  # 2 executed + 1 skipped

    saved_records = [call.kwargs for call in mock_repo.create.call_args_list]
    statuses = [r["status"] for r in saved_records]
    assert "success" in statuses
    assert "skipped" in statuses


def test_sync_account_called_after_execution():
    """실제 실행 완료 후 KisAccountSyncService.sync_account()가 1회 호출된다."""
    recommendations = [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=1,
            currency="KRW",
            quantity=Decimal("7"),
        ),
    ]

    mock_order_client = MagicMock()
    mock_order_client.place_order.return_value = {
        "rt_cd": "0",
        "msg_cd": "APBK0013",
        "msg1": "주문 전송 완료",
    }
    mock_sync_service = MagicMock()

    service = RebalanceExecutionService(
        order_client=mock_order_client,
        sync_service=mock_sync_service,
    )
    service.execute_rebalance_orders(recommendations, dry_run=False)

    mock_sync_service.sync_account.assert_called_once()


def test_sync_failure_preserves_results_and_exposes_warning():
    """동기화 실패 시 주문 결과는 유지하고 동기화 실패를 별도 경고로 노출한다."""
    recommendations = [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=1,
            currency="KRW",
            quantity=Decimal("7"),
        ),
    ]

    mock_order_client = MagicMock()
    mock_order_client.place_order.return_value = {
        "rt_cd": "0",
        "msg_cd": "APBK0013",
        "msg1": "주문 전송 완료",
    }
    mock_sync_service = MagicMock()
    mock_sync_service.sync_account.side_effect = RuntimeError("sync failed")

    service = RebalanceExecutionService(
        order_client=mock_order_client,
        sync_service=mock_sync_service,
    )
    result = service.execute_rebalance_orders(recommendations, dry_run=False)

    # Order results are preserved despite sync failure
    assert result.executions is not None
    assert len(result.executions) == 1
    assert result.executions[0].status == "success"

    # Sync failure exposed as warning
    assert result.sync_warning is not None
    assert "sync failed" in result.sync_warning
