FROM python:3.13-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PATH="/app/.venv/bin:${PATH}" \
    PYTHONPATH=/app/src

RUN groupadd --system app && useradd --system --create-home --gid app app

WORKDIR /app

RUN pip install --no-cache-dir uv

COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-dev --no-install-project

COPY src ./src

USER app

EXPOSE 8000

CMD ["uvicorn", "portfolio_manager.web.app:app", "--host", "0.0.0.0", "--port", "8000"]
