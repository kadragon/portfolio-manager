"""Unit tests for PortfolioInsightService."""

from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import date, datetime, timezone
from decimal import Decimal
from typing import Any
from uuid import UUID, uuid4

import pytest

from portfolio_manager.models import Account, Deposit, Group, Stock
from portfolio_manager.models.rebalance import RebalanceAction, RebalanceRecommendation
from portfolio_manager.services.llm.ollama_client import (
    ChatResponse,
    OllamaUnavailableError,
    ToolCall,
)
from portfolio_manager.services.portfolio_insight_service import (
    PortfolioInsightService,
)
from portfolio_manager.services.portfolio_service import (
    PortfolioSummary,
    StockHoldingWithPrice,
)
from portfolio_manager.services.rebalance_service import (
    GroupDiagnostic,
    RebalancePlan,
    RegionDiagnostic,
)


# --- fakes ------------------------------------------------------------------


class FakeOllamaClient:
    def __init__(self, responses: list[ChatResponse | Exception]) -> None:
        self._responses = list(responses)
        self.calls: list[dict[str, Any]] = []

    @property
    def model(self) -> str:
        return "fake"

    def chat(
        self,
        messages: list[dict[str, Any]],
        *,
        model: str | None = None,
        tools: list[dict[str, Any]] | None = None,
        format: str | dict[str, Any] | None = None,
        options: dict[str, Any] | None = None,
    ) -> ChatResponse:
        self.calls.append({"messages": messages, "tools": tools, "format": format})
        if not self._responses:
            raise AssertionError("No more canned responses for FakeOllamaClient")
        item = self._responses.pop(0)
        if isinstance(item, Exception):
            raise item
        return item


@dataclass
class FakePortfolioService:
    summary_with_rates: PortfolioSummary
    summary_without_rates: PortfolioSummary

    def get_portfolio_summary(
        self,
        *,
        include_change_rates: bool = True,
        change_rate_periods: tuple[str, ...] | None = None,
    ) -> PortfolioSummary:
        return (
            self.summary_with_rates
            if include_change_rates
            else self.summary_without_rates
        )


class FakeGroupRepository:
    def __init__(self, groups: list[Group]) -> None:
        self._groups = groups

    def list_all(self) -> list[Group]:
        return self._groups


class FakeStockRepository:
    def __init__(self, stocks: list[Stock]) -> None:
        self._stocks = stocks

    def list_all(self) -> list[Stock]:
        return self._stocks


class FakeAccountRepository:
    def __init__(self, accounts: list[Account]) -> None:
        self._accounts = accounts

    def list_all(self) -> list[Account]:
        return self._accounts


class FakeHoldingRepository:
    def list_by_account(self, account_id: UUID) -> list:  # noqa: ARG002
        return []


class FakeDepositRepository:
    def __init__(self, deposits: list[Deposit]) -> None:
        self._deposits = deposits

    def list_all(self) -> list[Deposit]:
        return self._deposits


class StubRebalanceService:
    def __init__(self, plan: RebalancePlan) -> None:
        self._plan = plan

    def build_plan(self, **_: Any) -> RebalancePlan:
        return self._plan


# --- fixtures ---------------------------------------------------------------


def _now() -> datetime:
    return datetime.now(timezone.utc)


@pytest.fixture
def group_kr_growth() -> Group:
    return Group(
        id=uuid4(),
        name="국내성장",
        created_at=_now(),
        updated_at=_now(),
        target_percentage=40.0,
    )


@pytest.fixture
def group_us_growth() -> Group:
    return Group(
        id=uuid4(),
        name="해외성장",
        created_at=_now(),
        updated_at=_now(),
        target_percentage=30.0,
    )


@pytest.fixture
def stock_samsung(group_kr_growth: Group) -> Stock:
    return Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group_kr_growth.id,
        created_at=_now(),
        updated_at=_now(),
        exchange=None,
        name="삼성전자",
    )


@pytest.fixture
def stock_tesla(group_us_growth: Group) -> Stock:
    return Stock(
        id=uuid4(),
        ticker="TSLA",
        group_id=group_us_growth.id,
        created_at=_now(),
        updated_at=_now(),
        exchange="NASD",
        name="Tesla",
    )


def _make_summary(
    *,
    group_kr: Group,
    group_us: Group,
    stock_samsung: Stock,
    stock_tesla: Stock,
    include_change_rates: bool,
) -> PortfolioSummary:
    samsung_holding = StockHoldingWithPrice(
        stock=stock_samsung,
        quantity=Decimal("10"),
        price=Decimal("70000"),
        currency="KRW",
        name="삼성전자",
        value_krw=Decimal("700000"),
        change_rates={"1d": Decimal("2.5")} if include_change_rates else None,
    )
    tesla_holding = StockHoldingWithPrice(
        stock=stock_tesla,
        quantity=Decimal("2"),
        price=Decimal("200"),
        currency="USD",
        name="Tesla",
        value_krw=Decimal("540000"),
        change_rates={"1d": Decimal("-3.0")} if include_change_rates else None,
    )
    return PortfolioSummary(
        holdings=[(group_kr, samsung_holding), (group_us, tesla_holding)],
        total_value=Decimal("1240000"),
        total_stock_value=Decimal("1240000"),
        total_cash_balance=Decimal("260000"),
        total_assets=Decimal("1500000"),
        total_invested=Decimal("1400000"),
        return_rate=Decimal("7.14"),
        first_deposit_date=date(2025, 1, 1),
        annualized_return_rate=Decimal("10.0"),
    )


@pytest.fixture
def service_factory(
    group_kr_growth: Group,
    group_us_growth: Group,
    stock_samsung: Stock,
    stock_tesla: Stock,
):
    def _build(
        *,
        ollama_responses: list[ChatResponse | Exception],
        plan: RebalancePlan | None = None,
    ) -> tuple[PortfolioInsightService, FakeOllamaClient]:
        summary_with = _make_summary(
            group_kr=group_kr_growth,
            group_us=group_us_growth,
            stock_samsung=stock_samsung,
            stock_tesla=stock_tesla,
            include_change_rates=True,
        )
        summary_without = _make_summary(
            group_kr=group_kr_growth,
            group_us=group_us_growth,
            stock_samsung=stock_samsung,
            stock_tesla=stock_tesla,
            include_change_rates=False,
        )
        ollama = FakeOllamaClient(ollama_responses)
        rebalance = StubRebalanceService(plan) if plan is not None else None
        deposit = Deposit(
            id=uuid4(),
            amount=Decimal("1400000"),
            deposit_date=date(2025, 1, 1),
            created_at=_now(),
            updated_at=_now(),
            note="초기 입금",
        )
        service = PortfolioInsightService(
            portfolio_service=FakePortfolioService(
                summary_with_rates=summary_with,
                summary_without_rates=summary_without,
            ),
            account_repository=FakeAccountRepository([]),
            holding_repository=FakeHoldingRepository(),
            group_repository=FakeGroupRepository([group_kr_growth, group_us_growth]),
            stock_repository=FakeStockRepository([stock_samsung, stock_tesla]),
            deposit_repository=FakeDepositRepository([deposit]),
            ollama_client=ollama,
            rebalance_service=rebalance,  # type: ignore[arg-type]
        )
        return service, ollama

    return _build


# --- narrative tests --------------------------------------------------------


def test_generate_narrative_builds_snapshot_and_sends_prompt(service_factory) -> None:
    service, ollama = service_factory(
        ollama_responses=[
            ChatResponse(content="일간 요약 텍스트", tool_calls=[], model="m")
        ],
    )

    result = service.generate_narrative(period="daily")

    assert result.narrative == "일간 요약 텍스트"
    assert result.error is None

    snapshot = result.snapshot
    assert snapshot.period == "daily"
    assert snapshot.rate_label == "1d"
    assert snapshot.total_assets == Decimal("1500000")
    # Samsung +2.5% on 700_000 = +17_500; Tesla -3.0% on 540_000 = -16_200
    assert len(snapshot.top_contributors) == 1
    assert snapshot.top_contributors[0].ticker == "005930"
    assert snapshot.top_contributors[0].contribution_krw == Decimal("17500.00")
    assert len(snapshot.bottom_contributors) == 1
    assert snapshot.bottom_contributors[0].ticker == "TSLA"

    # Group weights present for both configured groups
    names = {w.name for w in snapshot.group_weights}
    assert names == {"국내성장", "해외성장"}

    # Payload is wrapped in a ```json``` fence and numbers appear as strings
    user_msg = ollama.calls[0]["messages"][-1]["content"]
    assert "```json" in user_msg
    assert '"total_assets_krw": "1500000"' in user_msg
    assert '"rate_label": "1d"' in user_msg


def test_generate_narrative_returns_error_when_ollama_unavailable(
    service_factory,
) -> None:
    service, _ = service_factory(
        ollama_responses=[OllamaUnavailableError("연결 실패")],
    )

    result = service.generate_narrative(period="daily")

    assert result.narrative == ""
    assert result.error is not None
    assert "연결 실패" in result.error
    # Snapshot still populated for template rendering
    assert result.snapshot.total_assets == Decimal("1500000")


# --- rebalance XAI tests ----------------------------------------------------


def _make_plan_with_items() -> RebalancePlan:
    recs = [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.SELL,
            amount=Decimal("50000"),
            priority=1,
            currency="KRW",
            quantity=Decimal("1"),
            stock_name="삼성전자",
            group_name="국내성장",
            account_name="메인",
            rebalance_group_name="국내성장",
            reason="과열 그룹 절반 감축",
            trigger_type="group",
            amount_krw=Decimal("50000"),
        ),
    ]
    return RebalancePlan(
        sell_recommendations=recs,
        buy_recommendations=[],
        region_diagnostic=RegionDiagnostic(
            target_kr_percentage=Decimal("40"),
            target_us_percentage=Decimal("60"),
            current_kr_percentage=Decimal("45"),
            current_us_percentage=Decimal("55"),
            lower_kr_percentage=Decimal("35"),
            upper_kr_percentage=Decimal("45"),
            is_triggered=False,
        ),
        group_diagnostics=[
            GroupDiagnostic(
                rebalance_group_name="국내성장",
                target_percentage=Decimal("40"),
                band_percentage=Decimal("5"),
                lower_percentage=Decimal("35"),
                upper_percentage=Decimal("45"),
                current_percentage=Decimal("46"),
                current_value_krw=Decimal("700000"),
                is_upper_breached=True,
                is_lower_breached=False,
            ),
        ],
        total_assets_krw=Decimal("1500000"),
    )


def test_explain_rebalance_returns_empty_summary_when_plan_has_no_items(
    service_factory,
) -> None:
    plan = RebalancePlan(
        sell_recommendations=[],
        buy_recommendations=[],
        region_diagnostic=RegionDiagnostic(
            target_kr_percentage=Decimal("40"),
            target_us_percentage=Decimal("60"),
            current_kr_percentage=Decimal("40"),
            current_us_percentage=Decimal("60"),
            lower_kr_percentage=Decimal("35"),
            upper_kr_percentage=Decimal("45"),
            is_triggered=False,
        ),
        group_diagnostics=[],
        total_assets_krw=Decimal("1500000"),
    )
    service, ollama = service_factory(ollama_responses=[], plan=plan)

    result = service.explain_rebalance()

    assert result.summary.startswith("현재 리밸런싱")
    assert result.rationales == {}
    # No LLM call when there's nothing to explain
    assert ollama.calls == []


def test_explain_rebalance_parses_llm_json_response(service_factory) -> None:
    plan = _make_plan_with_items()
    llm_payload = json.dumps(
        {
            "summary": "국내성장 과열 해소가 필요합니다.",
            "items": [
                {"priority": 1, "rationale": "국내성장 상단 밴드 이탈로 감축 권고."}
            ],
        },
        ensure_ascii=False,
    )
    service, ollama = service_factory(
        ollama_responses=[ChatResponse(content=llm_payload)],
        plan=plan,
    )

    result = service.explain_rebalance()

    assert "국내성장 과열 해소" in result.summary
    assert result.rationales[1] == "국내성장 상단 밴드 이탈로 감축 권고."
    assert result.error is None
    # format=json must be requested
    assert ollama.calls[0]["format"] == "json"


def test_explain_rebalance_falls_back_to_python_reasons_on_parse_failure(
    service_factory,
) -> None:
    plan = _make_plan_with_items()
    service, _ = service_factory(
        ollama_responses=[ChatResponse(content="not-json-at-all")],
        plan=plan,
    )

    result = service.explain_rebalance()

    assert result.error is not None
    assert result.rationales[1] == "과열 그룹 절반 감축"
    assert result.summary == ""


def test_explain_rebalance_falls_back_when_ollama_unavailable(service_factory) -> None:
    plan = _make_plan_with_items()
    service, _ = service_factory(
        ollama_responses=[OllamaUnavailableError("timeout")],
        plan=plan,
    )

    result = service.explain_rebalance()

    assert result.error is not None
    assert "timeout" in result.error
    assert result.rationales[1] == "과열 그룹 절반 감축"


# --- Q&A tests --------------------------------------------------------------


def test_answer_question_rejects_empty_question(service_factory) -> None:
    service, ollama = service_factory(ollama_responses=[])

    result = service.answer_question("   ")

    assert result.error == "질문이 비어 있습니다."
    assert ollama.calls == []


def test_answer_question_returns_direct_answer_when_no_tool_calls(
    service_factory,
) -> None:
    service, _ = service_factory(
        ollama_responses=[ChatResponse(content="총자산은 150만원입니다.")],
    )

    result = service.answer_question("내 총자산은?")

    assert result.answer == "총자산은 150만원입니다."
    assert result.tool_calls_used == []
    assert result.error is None


def test_answer_question_executes_tool_and_feeds_result_back(service_factory) -> None:
    service, ollama = service_factory(
        ollama_responses=[
            ChatResponse(
                content="",
                tool_calls=[ToolCall(name="get_group_summary", arguments={})],
            ),
            ChatResponse(content="국내성장이 목표 대비 소폭 높습니다."),
        ],
    )

    result = service.answer_question("내 국내성장 비중은 어때?")

    assert result.answer == "국내성장이 목표 대비 소폭 높습니다."
    assert result.tool_calls_used == ["get_group_summary"]

    # Second call should include the tool result message
    second_messages = ollama.calls[1]["messages"]
    tool_messages = [m for m in second_messages if m.get("role") == "tool"]
    assert len(tool_messages) == 1
    tool_payload = json.loads(tool_messages[0]["content"])
    assert "groups" in tool_payload
    group_names = [g["name"] for g in tool_payload["groups"]]
    assert "국내성장" in group_names


def test_answer_question_uses_json_fallback_when_tool_loop_yields_no_content(
    service_factory,
) -> None:
    # First two responses: empty content, no tool_calls → exhausts loop + final chat.
    fallback_final = json.dumps(
        {"action": "final_answer", "text": "JSON 모드 답변"}, ensure_ascii=False
    )
    service, ollama = service_factory(
        ollama_responses=[
            ChatResponse(content="", tool_calls=[]),
            ChatResponse(content="", tool_calls=[]),
            ChatResponse(content=fallback_final),
        ],
    )

    result = service.answer_question("아무거나 물어볼게")

    assert result.answer == "JSON 모드 답변"
    assert result.error is None
    # Final fallback call must request json format
    assert ollama.calls[-1]["format"] == "json"


def test_answer_question_reports_error_on_ollama_failure_mid_flight(
    service_factory,
) -> None:
    service, _ = service_factory(
        ollama_responses=[OllamaUnavailableError("down")],
    )

    result = service.answer_question("질문")

    assert result.answer == ""
    assert result.error is not None
    assert "down" in result.error


# --- Q&A tool direct tests --------------------------------------------------


def test_tool_get_holding_value_matches_by_name_or_ticker(service_factory) -> None:
    service, _ = service_factory(ollama_responses=[])

    by_ticker = service._dispatch_qa_tool("get_holding_value", {"query": "TSLA"})
    by_name = service._dispatch_qa_tool("get_holding_value", {"query": "삼성"})

    assert by_ticker["ticker"] == "TSLA"
    assert by_name["ticker"] == "005930"


def test_tool_get_holding_value_returns_error_when_missing(service_factory) -> None:
    service, _ = service_factory(ollama_responses=[])

    result = service._dispatch_qa_tool("get_holding_value", {"query": "unknown"})

    assert "error" in result


def test_dispatch_unknown_tool_returns_error(service_factory) -> None:
    service, _ = service_factory(ollama_responses=[])

    result = service._dispatch_qa_tool("does_not_exist", {})

    assert "error" in result
