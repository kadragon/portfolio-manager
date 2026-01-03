# plan.md

# 기본 프로젝트 구축 계획 (Python + Rich 기반 CLI, 한국투자증권 API)

## 목표
- Python + Rich 기반의 CLI 주식관리 프로그램 구축
- 한국투자증권(KIS) API를 통해 한국/미국 주식 시세 수집
- 향후 기능 확장을 고려한 모듈형 아키텍처 설계

## 0. 의사결정(확인 필요)
## 0. 의사결정(확정)
- Python 버전: 최신 안정(stable) 버전 사용 (실제 설치 시점에 확인)
- 패키지 매니저: uv
- 데이터 저장: Supabase (Postgres)
- 배포 방식: 배포 없음, 로컬 실행용 (개인 프로젝트)

## 1. 초기 구조 설계
- 폴더 구조
  - src/
    - portfolio_manager/
      - cli/            # Rich 기반 CLI
      - core/           # 도메인 로직
      - services/       # KIS API 연동
      - storage/        # 로컬 저장소
      - config/         # 환경설정
  - tests/
  - docs/

## 2. 의존성 선정
- CLI 프레임워크: rich + typer
- HTTP 요청: httpx
- 환경변수: python-dotenv
- 테스트: pytest
- 데이터 저장: sqlite3 또는 SQLAlchemy

## 3. KIS API 연동 설계
- 인증 흐름 (Access Token 발급/갱신)
- 한국 주식 시세 조회
- 미국 주식 시세 조회
- 응답 파싱 및 공통 모델 정의

## 4. 핵심 기능 MVP
- 종목 등록/삭제
- 종목 목록 조회
- 시세 조회 (KR/US)
- 가격 히스토리 저장

## 5. 테스트 전략
- API 연동 Mock 테스트
- 핵심 도메인 로직 테스트
- CLI 명령어 단위 테스트

## 6. TDD 진행 방식
- 테스트 1개 작성 → 실패 확인 → 최소 구현 → 리팩터
- 구조적 변경과 행동 변경 분리
- 커밋 메시지에 structural/behavioral 구분

## 7. 문서화
- README: 설치/사용법/API 키 설정
- docs/: 설계 요약, API 연동 가이드

## 테스트 체크리스트 (TDD 순서)
- [x] Textual 앱 실행 시 기본 헤더(앱 이름)가 표시된다
- [x] Textual 앱에서 종목 목록 뷰가 빈 상태를 표시한다
- [x] KIS 인증 토큰을 저장/로드하는 서비스가 동작한다
- [x] KIS 한국 주식 현재가 조회가 공통 모델로 변환된다
- [x] KIS 미국 주식 현재가 조회가 공통 모델로 변환된다
- [x] 특정일 USD 환율 조회 로직이 동작한다

## 추가: KIS 실제 API 호출
- [x] KIS access token 요청이 /oauth2/tokenP로 POST 되며, 토큰 응답을 파싱한다
- [x] KIS 국내주식 기본정보 조회가 요청/파싱된다
- [x] KIS 국내주식 현재가 시세 조회가 요청/파싱된다
- [x] KIS 토큰이 유효하면 재사용하고, 만료 시 자동 재발급한다
- [x] KIS 해외주식 현재가(미국) 실호출 스크립트가 동작한다

---
## Next Action ("go" 시 시작)
- plan.md에서 아직 체크되지 않은 첫 테스트부터 작성
- TDD 사이클로 구현
