"""Route tests for /insights."""

from __future__ import annotations

from dataclasses import dataclass, field
from decimal import Decimal
from typing import Any

from portfolio_manager.services.portfolio_insight_service import (
    ContributorInfo,
    GroupWeightInfo,
    NarrativeResult,
    NarrativeSnapshot,
    QaResult,
    RebalanceExplanation,
)
from portfolio_manager.services.rebalance_service import RebalancePlan, RegionDiagnostic


def _make_snapshot(period: str = "daily") -> NarrativeSnapshot:
    return NarrativeSnapshot(
        period=period,
        rate_label="1d" if period == "daily" else "1m",
        total_assets=Decimal("1500000"),
        total_stock_value=Decimal("1240000"),
        total_cash_balance=Decimal("260000"),
        return_rate=Decimal("7.14"),
        top_contributors=[
            ContributorInfo(
                ticker="005930",
                name="삼성전자",
                group_name="국내성장",
                change_rate=Decimal("2.5"),
                value_krw=Decimal("700000"),
                contribution_krw=Decimal("17500"),
            )
        ],
        bottom_contributors=[],
        group_weights=[
            GroupWeightInfo(
                name="국내성장",
                current_percentage=Decimal("46"),
                target_percentage=Decimal("40"),
                diff_percentage=Decimal("6"),
            )
        ],
    )


def _empty_plan() -> RebalancePlan:
    return RebalancePlan(
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


@dataclass
class StubInsightService:
    narrative_result: NarrativeResult | None = None
    rebalance_result: RebalanceExplanation | None = None
    qa_result: QaResult | None = None
    raise_on: dict[str, Exception] = field(default_factory=dict)
    generate_calls: list[str] = field(default_factory=list)
    qa_calls: list[str] = field(default_factory=list)

    def generate_narrative(self, period: str = "daily") -> NarrativeResult:
        self.generate_calls.append(period)
        if "narrative" in self.raise_on:
            raise self.raise_on["narrative"]
        return self.narrative_result or NarrativeResult(
            snapshot=_make_snapshot(period), narrative="요약 텍스트"
        )

    def explain_rebalance(self) -> RebalanceExplanation:
        if "rebalance" in self.raise_on:
            raise self.raise_on["rebalance"]
        return self.rebalance_result or RebalanceExplanation(
            plan=_empty_plan(),
            summary="현재 리밸런싱이 필요한 편차가 없습니다.",
            rationales={},
        )

    def answer_question(self, question: str) -> QaResult:
        self.qa_calls.append(question)
        if "qa" in self.raise_on:
            raise self.raise_on["qa"]
        return self.qa_result or QaResult(
            question=question, answer="답변", tool_calls_used=[]
        )


def _install(fake_container, **kwargs: Any) -> StubInsightService:
    stub = StubInsightService(**kwargs)
    fake_container.portfolio_insight_service = stub
    return stub


def test_get_insights_returns_unavailable_when_service_missing(
    client, fake_container
) -> None:
    fake_container.portfolio_insight_service = None

    response = client.get("/insights")

    assert response.status_code == 200
    assert "AI 인사이트 서비스가 설정되지 않았습니다" in response.text


def test_get_insights_does_not_call_ollama_on_page_load(client, fake_container) -> None:
    stub = _install(fake_container)

    response = client.get("/insights")

    assert response.status_code == 200
    # Page shell must defer narrative generation to the HTMX partial request.
    assert stub.generate_calls == []
    # Full layout includes base nav (AI 인사이트 링크)
    assert "AI 인사이트" in response.text
    # Shell should set up the HTMX trigger that will fetch the narrative.
    assert "/insights/narrative?period=daily" in response.text


def test_get_narrative_partial_honors_period(client, fake_container) -> None:
    stub = _install(fake_container)

    response = client.get(
        "/insights/narrative?period=weekly", headers={"HX-Request": "true"}
    )

    assert response.status_code == 200
    assert stub.generate_calls == ["weekly"]
    # Partial must not contain the full base layout's <html> tag
    assert "<html" not in response.text.lower()


def test_get_narrative_falls_back_to_daily_for_unknown_period(
    client, fake_container
) -> None:
    stub = _install(fake_container)

    client.get("/insights/narrative?period=garbage")

    assert stub.generate_calls == ["daily"]


def test_get_narrative_returns_error_banner_on_exception(
    client, fake_container
) -> None:
    _install(fake_container, raise_on={"narrative": RuntimeError("boom")})

    response = client.get("/insights/narrative")

    assert response.status_code == 200
    assert "boom" in response.text


def test_get_rebalance_xai_returns_summary(client, fake_container) -> None:
    _install(fake_container)

    response = client.get("/insights/rebalance-xai")

    assert response.status_code == 200
    assert "현재 리밸런싱이 필요한 편차가 없습니다." in response.text


def test_get_rebalance_xai_returns_unavailable_when_service_missing(
    client, fake_container
) -> None:
    fake_container.portfolio_insight_service = None

    response = client.get("/insights/rebalance-xai")

    assert response.status_code == 200
    assert "AI 인사이트 서비스가 설정되지 않았습니다" in response.text


def test_post_qa_rejects_empty_question(client, fake_container) -> None:
    stub = _install(fake_container)

    response = client.post("/insights/qa", data={"question": "   "})

    assert response.status_code == 200
    assert "질문을 입력하세요" in response.text
    assert stub.qa_calls == []


def test_post_qa_renders_result(client, fake_container) -> None:
    stub = _install(
        fake_container,
        qa_result=QaResult(
            question="내 총자산은?",
            answer="총자산은 150만원입니다.",
            tool_calls_used=["get_group_summary"],
        ),
    )

    response = client.post("/insights/qa", data={"question": "내 총자산은?"})

    assert response.status_code == 200
    assert "총자산은 150만원입니다." in response.text
    assert "get_group_summary" in response.text
    assert stub.qa_calls == ["내 총자산은?"]


def test_post_qa_returns_unavailable_when_service_missing(
    client, fake_container
) -> None:
    fake_container.portfolio_insight_service = None

    response = client.post("/insights/qa", data={"question": "hi"})

    assert response.status_code == 200
    assert "AI 인사이트 서비스가 설정되지 않았습니다" in response.text


def test_post_qa_returns_error_on_service_exception(client, fake_container) -> None:
    _install(fake_container, raise_on={"qa": RuntimeError("crash")})

    response = client.post("/insights/qa", data={"question": "질문"})

    assert response.status_code == 200
    assert "crash" in response.text
