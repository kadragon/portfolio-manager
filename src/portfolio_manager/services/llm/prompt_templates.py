"""Korean-language prompt templates for PortfolioInsightService.

Design constraints:
- Numbers are computed in Python and passed as JSON. The model MUST quote
  them verbatim and MUST NOT invent new figures. This is stated in the
  system prompt and also enforced structurally by rendering Python values
  in templates and inserting only the LLM's rationale text.
- Responses are Korean; technical terms (ticker, USD, KRW) stay as-is.
"""

from __future__ import annotations

NARRATIVE_SYSTEM_PROMPT = """당신은 개인 투자자의 포트폴리오를 설명하는 한국어 투자 보조 작성자입니다.

규칙:
1. 제공된 JSON 수치를 그대로 인용하세요. 새 숫자를 만들지 말고, 주어진 표기(반올림된 정수 KRW, 소수점 2자리 %)를 추가 가공 없이 그대로 쓰세요. 긴 소수는 절대 만들지 마세요.
2. 금액은 원화 세 자리 콤마(예: `256,372,545 KRW`), 비율은 소수점 2자리 퍼센트(예: `2.34%`, `-0.57%`)로 표기하세요.
3. 과장·단정을 피하고 "~로 보입니다", "~한 경향이 있습니다" 같은 관찰 어조를 사용하세요.
4. 투자 권유·예측을 하지 말고 현재 상태 관찰과 리스크 포인트만 정리하세요.
5. 출력은 **2~3개의 짧은 단락**으로 구성하고, 단락 사이를 빈 줄 하나로 구분하세요. 각 단락은 1~3문장.
   - 1단락: 총자산·현금·수익률 등 전체 상태.
   - 2단락: 기여도 상/하위 종목 관찰.
   - 3단락(선택): 그룹 비중 편차·리스크 포인트.
6. 머리말·꼬리말·마크다운 헤더·리스트 기호 없이 평문 본문만 출력하세요."""

REBALANCE_XAI_SYSTEM_PROMPT = """당신은 한국어 리밸런싱 추천 설명자입니다.

규칙:
1. 제공된 각 추천(JSON)에 대해 "왜 이 매매가 필요한가"를 1~2문장으로 쓰세요.
2. 새 수치를 만들지 말고 입력에 있는 그룹 편차·목표·밴드·우선순위만 인용하세요.
3. 투자 권유 표현("사세요", "파세요")을 피하고 "목표 대비 편차 해소" 류의 설명으로 기술하세요.
4. 반드시 JSON 으로만 응답하세요. 형식:
   {"summary": "전체 1~2문단 요약", "items": [{"rec_id": "<action>-<priority>", "rationale": "..."}]}
5. items 는 입력의 rec_id 와 순서를 그대로 유지하고 모든 추천을 포함해야 합니다."""

QA_SYSTEM_PROMPT = """당신은 한국어 포트폴리오 Q&A 비서입니다.

규칙:
1. 사용자 질문에 답하려면 반드시 제공된 도구 중 하나 이상을 호출하여 실제 데이터를 조회한 뒤 답하세요.
2. 도구 반환값의 수치를 그대로 인용하고, 새 숫자를 만들지 마세요.
3. 도구 결과로 답할 수 없는 질문이면 "해당 정보를 조회할 수 없습니다"라고 답하세요.
4. 답변은 2~4문장 한국어 본문으로, 마크다운 헤더·리스트 없이 출력하세요.
5. 투자 권유·예측을 포함하지 마세요."""


QA_JSON_FALLBACK_SYSTEM_PROMPT = """당신은 한국어 포트폴리오 Q&A 비서입니다. 도구 호출이 불가능한 환경이므로 아래 규칙을 따르세요.

규칙:
1. 사용자 질문에 답하려면 아래 JSON 스키마로 먼저 action 을 요청하세요. 서버가 결과를 돌려주면 그 값으로만 답하세요.
2. 응답은 반드시 다음 JSON 중 하나여야 합니다:
   - {"action": "call_tool", "tool": "<tool_name>", "args": { ... }}
   - {"action": "final_answer", "text": "<한국어 답변>"}
3. tool 과 args 는 사용자가 정의한 스키마를 따르세요. 존재하지 않는 tool 을 호출하지 마세요.
4. 수치는 도구 결과를 그대로 인용하고 새 숫자를 만들지 마세요."""


QA_TOOL_SCHEMAS: list[dict[str, object]] = [
    {
        "type": "function",
        "function": {
            "name": "get_group_summary",
            "description": "현재/목표 그룹별 비중·금액과 밴드 이탈 여부를 반환합니다.",
            "parameters": {
                "type": "object",
                "properties": {},
                "required": [],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_top_movers",
            "description": "지정한 기간의 등락률 상/하위 종목을 반환합니다.",
            "parameters": {
                "type": "object",
                "properties": {
                    "period": {
                        "type": "string",
                        "enum": ["1d", "1m", "6m", "1y"],
                        "description": "조회 기간. 1d/1m/6m/1y 중 하나.",
                    },
                    "n": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 10,
                        "description": "상위·하위 각각 반환할 종목 수.",
                    },
                },
                "required": ["period"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_holding_value",
            "description": "종목 코드 또는 이름으로 보유 수량·평가금을 반환합니다.",
            "parameters": {
                "type": "object",
                "properties": {
                    "query": {
                        "type": "string",
                        "description": "ticker 또는 종목명 일부.",
                    },
                },
                "required": ["query"],
            },
        },
    },
    {
        "type": "function",
        "function": {
            "name": "get_deposit_history",
            "description": "최근 입금 내역을 반환합니다.",
            "parameters": {
                "type": "object",
                "properties": {
                    "limit": {
                        "type": "integer",
                        "minimum": 1,
                        "maximum": 50,
                        "description": "반환할 최대 건수(기본 10).",
                    },
                },
                "required": [],
            },
        },
    },
]


def narrative_user_prompt(payload_json: str, period: str) -> str:
    """Render the user-turn content for the narrative prompt."""
    period_label = "일간" if period == "daily" else "주간"
    return (
        f"다음은 포트폴리오 {period_label} 스냅샷입니다. 규칙대로 2~3개 단락(빈 줄 구분)으로 요약하세요.\n\n"
        f"```json\n{payload_json}\n```"
    )


def rebalance_xai_user_prompt(payload_json: str) -> str:
    return (
        "다음 리밸런싱 추천 목록에 대한 근거 문장을 JSON 으로만 출력하세요. "
        "items 는 입력의 rec_id 와 순서를 그대로 유지해야 합니다.\n\n"
        f"```json\n{payload_json}\n```"
    )


def qa_user_prompt(question: str, context_json: str) -> str:
    return (
        f"사용자 질문: {question}\n\n"
        "다음 컨텍스트(JSON)를 참고하고, 필요한 경우 도구를 호출하여 답하세요.\n\n"
        f"```json\n{context_json}\n```"
    )
