"""PortfolioInsightService — narrative, rebalance XAI, and Q&A backed by Ollama.

Design rule: Python owns every number. The LLM receives structured JSON and
returns Korean prose only. Templates render Python-computed values directly
and insert LLM text as `narrative` / `rationale` strings — the LLM cannot
move a digit through this surface.
"""

from __future__ import annotations

import json
import logging
import time
from collections import defaultdict
from dataclasses import dataclass, field
from decimal import ROUND_HALF_UP, Decimal
from typing import Any, Literal

from portfolio_manager.models.rebalance import (
    RebalanceAction,
    RebalanceRecommendation,
)
from portfolio_manager.repositories.account_repository import AccountRepository
from portfolio_manager.repositories.deposit_repository import DepositRepository
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository
from portfolio_manager.repositories.stock_repository import StockRepository
from portfolio_manager.services.llm.ollama_client import (
    OllamaClient,
    OllamaUnavailableError,
)
from portfolio_manager.services.llm.prompt_templates import (
    NARRATIVE_SYSTEM_PROMPT,
    QA_JSON_FALLBACK_SYSTEM_PROMPT,
    QA_SYSTEM_PROMPT,
    QA_TOOL_SCHEMAS,
    REBALANCE_XAI_SYSTEM_PROMPT,
    narrative_user_prompt,
    qa_user_prompt,
    rebalance_xai_user_prompt,
)
from portfolio_manager.services.portfolio_service import PortfolioService
from portfolio_manager.services.rebalance_service import RebalancePlan, RebalanceService

logger = logging.getLogger(__name__)

NarrativePeriod = Literal["daily", "weekly"]
_PERIOD_RATE_LABEL: dict[str, str] = {"daily": "1d", "weekly": "1m"}
_MAX_CONTRIBUTORS = 3
_QA_TOOL_ITERATIONS = 3
# soft cap; last in-flight call may add up to one chat timeout
_QA_DEADLINE_SEC: float = 120.0


# --- Result data classes -----------------------------------------------------


@dataclass
class ContributorInfo:
    ticker: str
    name: str
    group_name: str
    change_rate: Decimal
    value_krw: Decimal
    contribution_krw: Decimal


@dataclass
class GroupWeightInfo:
    name: str
    current_percentage: Decimal
    target_percentage: Decimal
    diff_percentage: Decimal


@dataclass
class NarrativeSnapshot:
    period: str
    rate_label: str
    total_assets: Decimal
    total_stock_value: Decimal
    total_cash_balance: Decimal
    return_rate: Decimal | None
    top_contributors: list[ContributorInfo]
    bottom_contributors: list[ContributorInfo]
    group_weights: list[GroupWeightInfo]


@dataclass
class NarrativeResult:
    snapshot: NarrativeSnapshot
    narrative: str
    error: str | None = None


@dataclass
class RebalanceExplanation:
    plan: RebalancePlan
    summary: str
    rationales: dict[str, str]
    error: str | None = None


@dataclass
class QaResult:
    question: str
    answer: str
    tool_calls_used: list[str] = field(default_factory=list)
    error: str | None = None


# --- Service -----------------------------------------------------------------


class PortfolioInsightService:
    """Aggregate portfolio context and hand it to Ollama for prose generation."""

    def __init__(
        self,
        *,
        portfolio_service: PortfolioService,
        account_repository: AccountRepository,
        holding_repository: HoldingRepository,
        group_repository: GroupRepository,
        stock_repository: StockRepository,
        deposit_repository: DepositRepository,
        ollama_client: OllamaClient,
        rebalance_service: RebalanceService | None = None,
    ) -> None:
        self._portfolio_service = portfolio_service
        self._account_repository = account_repository
        self._holding_repository = holding_repository
        self._group_repository = group_repository
        self._stock_repository = stock_repository
        self._deposit_repository = deposit_repository
        self._ollama = ollama_client
        self._rebalance_service = rebalance_service or RebalanceService()

    # -- narrative -----------------------------------------------------------

    def generate_narrative(self, period: NarrativePeriod = "daily") -> NarrativeResult:
        rate_label = _PERIOD_RATE_LABEL.get(period, "1d")
        snapshot = self._build_narrative_snapshot(period=period, rate_label=rate_label)
        payload = _snapshot_to_payload(snapshot)
        payload_json = json.dumps(payload, ensure_ascii=False, default=_json_default)

        try:
            response = self._ollama.chat(
                [
                    {"role": "system", "content": NARRATIVE_SYSTEM_PROMPT},
                    {
                        "role": "user",
                        "content": narrative_user_prompt(payload_json, period=period),
                    },
                ]
            )
        except OllamaUnavailableError as exc:
            logger.warning("Ollama unavailable for narrative: %s", exc)
            return NarrativeResult(snapshot=snapshot, narrative="", error=str(exc))

        return NarrativeResult(snapshot=snapshot, narrative=response.content)

    def _build_narrative_snapshot(
        self, *, period: str, rate_label: str
    ) -> NarrativeSnapshot:
        summary = self._portfolio_service.get_portfolio_summary(
            include_change_rates=True,
            change_rate_periods=(rate_label,),
        )

        contributors: list[ContributorInfo] = []
        value_by_group: dict[str, Decimal] = defaultdict(Decimal)

        for group, holding in summary.holdings:
            if holding.value_krw is not None:
                value_by_group[group.name] += holding.value_krw

            change = Decimal("0")
            if holding.change_rates:
                change = holding.change_rates.get(rate_label, Decimal("0"))
            if holding.value_krw is None:
                continue
            contribution = holding.value_krw * change / Decimal("100")
            contributors.append(
                ContributorInfo(
                    ticker=holding.stock.ticker,
                    name=holding.name or holding.stock.ticker,
                    group_name=group.name,
                    change_rate=change,
                    value_krw=holding.value_krw,
                    contribution_krw=contribution,
                )
            )

        positive = sorted(
            (c for c in contributors if c.contribution_krw > 0),
            key=lambda c: c.contribution_krw,
            reverse=True,
        )[:_MAX_CONTRIBUTORS]
        negative = sorted(
            (c for c in contributors if c.contribution_krw < 0),
            key=lambda c: c.contribution_krw,
        )[:_MAX_CONTRIBUTORS]

        total_stock = summary.total_stock_value
        group_weights: list[GroupWeightInfo] = []
        for group in self._group_repository.list_all():
            current_value = value_by_group.get(group.name, Decimal("0"))
            current_pct = (
                current_value / total_stock * Decimal("100")
                if total_stock > 0
                else Decimal("0")
            )
            target_pct = Decimal(str(group.target_percentage))
            group_weights.append(
                GroupWeightInfo(
                    name=group.name,
                    current_percentage=current_pct,
                    target_percentage=target_pct,
                    diff_percentage=current_pct - target_pct,
                )
            )

        return NarrativeSnapshot(
            period=period,
            rate_label=rate_label,
            total_assets=summary.total_assets,
            total_stock_value=summary.total_stock_value,
            total_cash_balance=summary.total_cash_balance,
            return_rate=summary.return_rate,
            top_contributors=positive,
            bottom_contributors=negative,
            group_weights=group_weights,
        )

    # -- rebalance XAI -------------------------------------------------------

    def explain_rebalance(self) -> RebalanceExplanation:
        plan = self._build_rebalance_plan()
        items = plan.sell_recommendations + plan.buy_recommendations

        if not items:
            return RebalanceExplanation(
                plan=plan,
                summary="현재 리밸런싱이 필요한 편차가 없습니다.",
                rationales={},
            )

        payload = _rebalance_plan_to_payload(plan)
        payload_json = json.dumps(payload, ensure_ascii=False, default=_json_default)

        try:
            response = self._ollama.chat(
                [
                    {"role": "system", "content": REBALANCE_XAI_SYSTEM_PROMPT},
                    {
                        "role": "user",
                        "content": rebalance_xai_user_prompt(payload_json),
                    },
                ],
                format="json",
            )
        except OllamaUnavailableError as exc:
            logger.warning("Ollama unavailable for rebalance XAI: %s", exc)
            return RebalanceExplanation(
                plan=plan,
                summary="",
                rationales=_fallback_rationales(items),
                error=str(exc),
            )

        parsed = _parse_rebalance_response(response.content)
        if parsed is None:
            return RebalanceExplanation(
                plan=plan,
                summary="",
                rationales=_fallback_rationales(items),
                error="LLM 응답을 파싱하지 못했습니다.",
            )

        summary_text, rationales = parsed
        # Ensure every recommendation has at least the Python-computed reason
        # as fallback when the LLM skips one.
        fallback = _fallback_rationales(items)
        for rec_id, reason in fallback.items():
            rationales.setdefault(rec_id, reason)

        return RebalanceExplanation(
            plan=plan,
            summary=summary_text,
            rationales=rationales,
        )

    def _build_rebalance_plan(self) -> RebalancePlan:
        summary = self._portfolio_service.get_portfolio_summary(
            include_change_rates=False
        )
        accounts = self._account_repository.list_all()
        holdings_by_account = {
            account.id: self._holding_repository.list_by_account(account.id)
            for account in accounts
        }
        groups = self._group_repository.list_all()
        stocks = self._stock_repository.list_all()
        return self._rebalance_service.build_plan(
            summary=summary,
            accounts=accounts,
            holdings_by_account=holdings_by_account,
            groups=groups,
            stocks=stocks,
        )

    # -- Q&A -----------------------------------------------------------------

    def answer_question(self, question: str) -> QaResult:
        question = question.strip()
        if not question:
            return QaResult(
                question=question,
                answer="",
                error="질문이 비어 있습니다.",
            )

        context = self._build_qa_context()
        context_json = json.dumps(context, ensure_ascii=False, default=_json_default)

        messages: list[dict[str, Any]] = [
            {"role": "system", "content": QA_SYSTEM_PROMPT},
            {"role": "user", "content": qa_user_prompt(question, context_json)},
        ]

        tools_used: list[str] = []
        _deadline = time.monotonic() + _QA_DEADLINE_SEC
        try:
            for _ in range(_QA_TOOL_ITERATIONS):
                if time.monotonic() > _deadline:
                    break
                response = self._ollama.chat(messages, tools=QA_TOOL_SCHEMAS)
                if not response.tool_calls:
                    if response.content:
                        return QaResult(
                            question=question,
                            answer=response.content,
                            tool_calls_used=tools_used,
                        )
                    break

                messages.append(
                    {
                        "role": "assistant",
                        "content": response.content,
                        "tool_calls": [
                            {
                                "function": {
                                    "name": call.name,
                                    "arguments": call.arguments,
                                }
                            }
                            for call in response.tool_calls
                        ],
                    }
                )
                for call in response.tool_calls:
                    tools_used.append(call.name)
                    result = self._dispatch_qa_tool(call.name, call.arguments)
                    messages.append(
                        {
                            "role": "tool",
                            "content": json.dumps(
                                result, ensure_ascii=False, default=_json_default
                            ),
                        }
                    )
            # Fallthrough: ask for a final answer without tools.
            if time.monotonic() > _deadline:
                return QaResult(
                    question=question,
                    answer="",
                    tool_calls_used=tools_used,
                    error=f"Q&A timed out after {_QA_DEADLINE_SEC:.0f}s",
                )
            final = self._ollama.chat(messages)
            if final.content:
                return QaResult(
                    question=question,
                    answer=final.content,
                    tool_calls_used=tools_used,
                )
        except OllamaUnavailableError as exc:
            logger.warning("Ollama unavailable for Q&A: %s", exc)
            return QaResult(
                question=question,
                answer="",
                tool_calls_used=tools_used,
                error=str(exc),
            )

        # Fallback: retry with JSON-action protocol.
        return self._answer_via_json_fallback(
            question, context_json, tools_used, _deadline
        )

    def _answer_via_json_fallback(
        self, question: str, context_json: str, tools_used: list[str], deadline: float
    ) -> QaResult:
        system_content = (
            QA_JSON_FALLBACK_SYSTEM_PROMPT
            + "\n\n사용 가능한 도구:\n"
            + _format_tool_schemas_for_prompt(QA_TOOL_SCHEMAS)
        )
        messages: list[dict[str, Any]] = [
            {"role": "system", "content": system_content},
            {"role": "user", "content": qa_user_prompt(question, context_json)},
        ]
        try:
            for _ in range(_QA_TOOL_ITERATIONS):
                if time.monotonic() > deadline:
                    return QaResult(
                        question=question,
                        answer="",
                        tool_calls_used=tools_used,
                        error=f"Q&A timed out after {_QA_DEADLINE_SEC:.0f}s",
                    )
                response = self._ollama.chat(messages, format="json")
                parsed = _safe_json_loads(response.content)
                if parsed is None:
                    break
                action = parsed.get("action")
                if action == "final_answer":
                    return QaResult(
                        question=question,
                        answer=str(parsed.get("text") or ""),
                        tool_calls_used=tools_used,
                    )
                if action == "call_tool":
                    tool = str(parsed.get("tool") or "")
                    args = parsed.get("args") or {}
                    if not isinstance(args, dict):
                        args = {}
                    tools_used.append(tool)
                    result = self._dispatch_qa_tool(tool, args)
                    messages.append({"role": "assistant", "content": response.content})
                    messages.append(
                        {
                            "role": "user",
                            "content": "도구 실행 결과: "
                            + json.dumps(
                                result, ensure_ascii=False, default=_json_default
                            ),
                        }
                    )
                    continue
                break
        except OllamaUnavailableError as exc:
            return QaResult(
                question=question,
                answer="",
                tool_calls_used=tools_used,
                error=str(exc),
            )

        return QaResult(
            question=question,
            answer="",
            tool_calls_used=tools_used,
            error="답변을 생성하지 못했습니다.",
        )

    def _build_qa_context(self) -> dict[str, Any]:
        summary = self._portfolio_service.get_portfolio_summary(
            include_change_rates=False
        )
        return {
            "total_assets_krw": summary.total_assets,
            "total_stock_value_krw": summary.total_stock_value,
            "total_cash_balance_krw": summary.total_cash_balance,
            "holdings_count": len(summary.holdings),
        }

    def _dispatch_qa_tool(self, name: str, arguments: dict[str, Any]) -> dict[str, Any]:
        try:
            if name == "get_group_summary":
                return self._tool_group_summary()
            if name == "get_top_movers":
                period = str(arguments.get("period") or "1d")
                n = int(arguments.get("n") or 3)
                return self._tool_top_movers(period=period, n=n)
            if name == "get_holding_value":
                query = str(arguments.get("query") or "")
                return self._tool_holding_value(query)
            if name == "get_deposit_history":
                limit = int(arguments.get("limit") or 10)
                return self._tool_deposit_history(limit=limit)
        except Exception as exc:  # noqa: BLE001
            logger.exception("QA tool %s failed", name)
            return {"error": f"도구 실행 실패: {exc}"}
        return {"error": f"알 수 없는 도구: {name}"}

    # -- QA tool implementations --------------------------------------------

    def _tool_group_summary(self) -> dict[str, Any]:
        summary = self._portfolio_service.get_portfolio_summary(
            include_change_rates=False
        )
        value_by_group: dict[str, Decimal] = defaultdict(Decimal)
        for group, holding in summary.holdings:
            if holding.value_krw is not None:
                value_by_group[group.name] += holding.value_krw

        total_stock = summary.total_stock_value
        groups = []
        for group in self._group_repository.list_all():
            value = value_by_group.get(group.name, Decimal("0"))
            current_pct = (
                value / total_stock * Decimal("100")
                if total_stock > 0
                else Decimal("0")
            )
            target_pct = Decimal(str(group.target_percentage))
            groups.append(
                {
                    "name": group.name,
                    "current_value_krw": value,
                    "current_percentage": current_pct,
                    "target_percentage": target_pct,
                    "diff_percentage": current_pct - target_pct,
                }
            )
        return {"groups": groups, "total_stock_value_krw": total_stock}

    def _tool_top_movers(self, *, period: str, n: int) -> dict[str, Any]:
        summary = self._portfolio_service.get_portfolio_summary(
            include_change_rates=True,
            change_rate_periods=(period,),
        )
        rows: list[dict[str, Any]] = []
        for group, holding in summary.holdings:
            change = Decimal("0")
            if holding.change_rates:
                change = holding.change_rates.get(period, Decimal("0"))
            rows.append(
                {
                    "ticker": holding.stock.ticker,
                    "name": holding.name or holding.stock.ticker,
                    "group_name": group.name,
                    "change_rate": change,
                }
            )
        rows.sort(key=lambda r: Decimal(str(r["change_rate"])), reverse=True)
        limit = max(1, min(n, 10))
        return {
            "period": period,
            "top": rows[:limit],
            "bottom": list(reversed(rows[-limit:])),
        }

    def _tool_holding_value(self, query: str) -> dict[str, Any]:
        query = query.strip().lower()
        if not query:
            return {"error": "query 가 비어 있습니다."}
        summary = self._portfolio_service.get_portfolio_summary(
            include_change_rates=False
        )
        for group, holding in summary.holdings:
            ticker = holding.stock.ticker.lower()
            name = (holding.name or "").lower()
            if query in ticker or (name and query in name):
                return {
                    "ticker": holding.stock.ticker,
                    "name": holding.name or holding.stock.ticker,
                    "group_name": group.name,
                    "quantity": holding.quantity,
                    "price_local": holding.price,
                    "currency": holding.currency,
                    "value_krw": holding.value_krw,
                }
        return {"error": f"'{query}' 에 해당하는 보유 종목을 찾지 못했습니다."}

    def _tool_deposit_history(self, *, limit: int) -> dict[str, Any]:
        limit = max(1, min(limit, 50))
        deposits = self._deposit_repository.list_all()[:limit]
        return {
            "deposits": [
                {
                    "amount_krw": d.amount,
                    "deposit_date": d.deposit_date.isoformat(),
                    "note": d.note,
                }
                for d in deposits
            ],
        }


# --- helpers ----------------------------------------------------------------


def _json_default(value: Any) -> Any:
    if isinstance(value, Decimal):
        # Preserve precision but drop trailing zeros for readability.
        text = format(value, "f")
        if "." in text:
            text = text.rstrip("0").rstrip(".")
        return text or "0"
    raise TypeError(f"Object of type {type(value).__name__} is not JSON serializable")


_KRW_Q = Decimal("1")
_PCT_Q = Decimal("0.01")


def _q_krw(value: Decimal | None) -> Decimal | None:
    if value is None:
        return None
    return Decimal(value).quantize(_KRW_Q, rounding=ROUND_HALF_UP)


def _q_pct(value: Decimal | None) -> Decimal | None:
    if value is None:
        return None
    return Decimal(value).quantize(_PCT_Q, rounding=ROUND_HALF_UP)


def _contributor_payload(c: ContributorInfo) -> dict[str, Any]:
    return {
        "ticker": c.ticker,
        "name": c.name,
        "group_name": c.group_name,
        "change_rate_percent": _q_pct(c.change_rate),
        "value_krw": _q_krw(c.value_krw),
        "contribution_krw": _q_krw(c.contribution_krw),
    }


def _group_weight_payload(g: GroupWeightInfo) -> dict[str, Any]:
    return {
        "name": g.name,
        "current_percentage": _q_pct(g.current_percentage),
        "target_percentage": _q_pct(g.target_percentage),
        "diff_percentage": _q_pct(g.diff_percentage),
    }


def _snapshot_to_payload(snapshot: NarrativeSnapshot) -> dict[str, Any]:
    return {
        "period": snapshot.period,
        "rate_label": snapshot.rate_label,
        "number_format_note": "금액은 반올림된 정수 KRW, 비율은 소수점 2자리 %. 그대로 인용하세요.",
        "totals": {
            "total_assets_krw": _q_krw(snapshot.total_assets),
            "total_stock_value_krw": _q_krw(snapshot.total_stock_value),
            "total_cash_balance_krw": _q_krw(snapshot.total_cash_balance),
            "return_rate_percent": _q_pct(snapshot.return_rate),
        },
        "top_contributors": [
            _contributor_payload(c) for c in snapshot.top_contributors
        ],
        "bottom_contributors": [
            _contributor_payload(c) for c in snapshot.bottom_contributors
        ],
        "group_weights": [_group_weight_payload(g) for g in snapshot.group_weights],
    }


def _rebalance_plan_to_payload(plan: RebalancePlan) -> dict[str, Any]:
    items = plan.sell_recommendations + plan.buy_recommendations
    return {
        "region": {
            "target_kr_percentage": plan.region_diagnostic.target_kr_percentage,
            "current_kr_percentage": plan.region_diagnostic.current_kr_percentage,
            "lower_kr_percentage": plan.region_diagnostic.lower_kr_percentage,
            "upper_kr_percentage": plan.region_diagnostic.upper_kr_percentage,
            "is_triggered": plan.region_diagnostic.is_triggered,
        },
        "group_diagnostics": [
            {
                "name": d.rebalance_group_name,
                "target_percentage": d.target_percentage,
                "current_percentage": d.current_percentage,
                "lower_percentage": d.lower_percentage,
                "upper_percentage": d.upper_percentage,
                "is_upper_breached": d.is_upper_breached,
                "is_lower_breached": d.is_lower_breached,
            }
            for d in plan.group_diagnostics or []
        ],
        "recommendations": [
            {
                "rec_id": (
                    rec.action.value
                    if isinstance(rec.action, RebalanceAction)
                    else str(rec.action)
                )
                + f"-{rec.priority}",
                "priority": rec.priority,
                "action": rec.action.value
                if isinstance(rec.action, RebalanceAction)
                else str(rec.action),
                "ticker": rec.ticker,
                "stock_name": rec.stock_name,
                "account_name": rec.account_name,
                "rebalance_group_name": rec.rebalance_group_name,
                "trigger_type": rec.trigger_type,
                "amount_krw": rec.amount_krw,
                "reason": rec.reason,
            }
            for rec in items
        ],
    }


def _parse_rebalance_response(content: str) -> tuple[str, dict[str, str]] | None:
    data = _safe_json_loads(content)
    if not isinstance(data, dict):
        return None
    summary_text = str(data.get("summary") or "").strip()
    items = data.get("items")
    rationales: dict[str, str] = {}
    if isinstance(items, list):
        for entry in items:
            if not isinstance(entry, dict):
                continue
            rec_id_raw = entry.get("rec_id")
            if not rec_id_raw:
                continue
            rec_id = str(rec_id_raw).strip()
            if not rec_id:
                continue
            rationale = str(entry.get("rationale") or "").strip()
            if rationale:
                rationales[rec_id] = rationale
    if not summary_text and not rationales:
        return None
    return summary_text, rationales


def _fallback_rationales(
    items: list[RebalanceRecommendation],
) -> dict[str, str]:
    result: dict[str, str] = {}
    for rec in items:
        if not rec.reason:
            continue
        action_value = (
            rec.action.value
            if isinstance(rec.action, RebalanceAction)
            else str(rec.action)
        )
        result[f"{action_value}-{rec.priority}"] = rec.reason
    return result


def _format_tool_schemas_for_prompt(schemas: list[dict[str, Any]]) -> str:
    """Render tool schemas as a compact textual listing for JSON-fallback prompts."""
    lines: list[str] = []
    for schema in schemas:
        fn = schema.get("function") if isinstance(schema, dict) else None
        if not isinstance(fn, dict):
            continue
        name = fn.get("name") or ""
        description = fn.get("description") or ""
        parameters = fn.get("parameters") or {}
        args_json = json.dumps(parameters, ensure_ascii=False)
        lines.append(f"- {name}(args): {description} args: {args_json}")
    return "\n".join(lines)


def _safe_json_loads(text: str) -> Any:
    if not text:
        return None
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        # Attempt to extract first balanced JSON object.
        start = text.find("{")
        end = text.rfind("}")
        if start != -1 and end != -1 and end > start:
            try:
                return json.loads(text[start : end + 1])
            except json.JSONDecodeError:
                return None
        return None
