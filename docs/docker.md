# Docker 실행 가이드 (온디맨드 개발 모드)

이 프로젝트는 로컬에서 필요할 때만 컨테이너를 올리고, `http://localhost:8000`으로 바로 접속하는 흐름을 기준으로 구성되어 있습니다.

## 1) 최초 1회 빌드 + 실행

```bash
docker compose up -d --build web
```

Linux에서 Docker와 로컬 실행을 번갈아 사용할 경우 `.data` 권한 충돌 방지를 위해 호스트 UID/GID를 함께 지정하세요.

```bash
LOCAL_UID=$(id -u) LOCAL_GID=$(id -g) docker compose up -d --build web
```

## 2) 이후 빠른 재개 / 중지

```bash
docker compose start web
docker compose stop web
```

## 3) 상태/로그 확인

```bash
docker compose ps
docker compose logs -f web
```

## 4) 접속 주소

- 웹: `http://localhost:8000`

## 5) 구성 요약

- 서비스명: `web`
- 포트: `127.0.0.1:8000:8000` (로컬 머신에서만 접근)
- 실행 사용자: `${LOCAL_UID:-1000}:${LOCAL_GID:-1000}` (호스트 UID/GID 매핑)
- 이미지 기본 사용자: `app` (non-root)
- 환경변수: `env_file: .env`
- 코드 마운트: `./src:/app/src` (코드 변경 시 `--reload`로 자동 반영)
- 토큰 캐시 마운트: `./.data:/app/.data` (KIS 토큰 파일 유지)
- 재시작 정책: `restart: "no"` (온디맨드 수동 실행)

## 6) 트러블슈팅

### `docker compose config`에서 `.env` 관련 오류

- 프로젝트 루트에 `.env`가 있어야 합니다.
- 최소 `SUPABASE_URL`, `SUPABASE_KEY`를 확인하세요.

### `http://localhost:8000` 접속 불가

```bash
docker compose ps
docker compose logs -f web
```

- `web` 상태가 `Up`인지 확인합니다.
- 포트 `8000`을 다른 프로세스가 사용 중이면 해당 프로세스를 종료하거나 포트를 변경하세요.

### 코드 변경이 반영되지 않음

- `src` 하위 파일을 수정했는지 확인하세요. 현재 마운트 대상은 `./src`입니다.
- 컨테이너 로그에서 `reload` 이벤트를 확인하세요.

### `.data/kis_token.json` 권한 오류

- Linux에서는 컨테이너 생성 시 UID/GID를 지정해 실행하세요.
- 이미 루트 소유 파일이 생겼다면 권한을 정리한 뒤 재시작하세요.

```bash
sudo chown -R "$(id -u):$(id -g)" .data
LOCAL_UID=$(id -u) LOCAL_GID=$(id -g) docker compose up -d --build web
```
