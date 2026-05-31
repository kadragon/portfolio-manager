# Docker 실행 가이드 (온디맨드 개발 모드)

이 프로젝트는 로컬에서 필요할 때만 컨테이너를 올리고, `http://localhost:8000`으로 바로 접속하는 흐름을 기준으로 구성되어 있습니다. 멀티스테이지 빌드로 `CGO_ENABLED=0` 정적 바이너리를 만들어 `alpine` 런타임에서 실행합니다.

## 1) 최초 1회 빌드 + 실행

```bash
docker compose up -d --build web
```

`.data` 볼륨에 SQLite DB(`portfolio.db`)가 영속됩니다. `.env`로 KIS/환율 설정을 주입합니다.

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

## 4) 코드 수정 반영

```bash
docker compose up -d --build web   # 코드 변경 후 재빌드
```

정적 바이너리이므로 코드 변경은 재빌드가 필요합니다 (소스 볼륨 마운트·핫리로드 없음). 빠른 로컬 개발은 `make dev`(네이티브 `go run` + CSS watch)를 권장합니다.
