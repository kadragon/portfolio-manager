"""Structural tests enforcing architecture layer boundaries.

Golden Principles enforced:
- GP-1: Repository layer owns all DB access (no raw ORM queries in web/ or services/)
- GP-3: Layer dependency direction (web → services → repositories → models; no reverse)
"""

import re
import pathlib

ROOT = pathlib.Path(__file__).parent.parent.parent / "src" / "portfolio_manager"

# Peewee model class-level query methods anchored to known ORM model class names.
# Anchoring prevents false positives on stdlib/framework classes (e.g. Response.delete).
_ORM_CLASS_QUERY_PATTERN = re.compile(
    r"\b(BaseModel|GroupModel|StockModel|AccountModel|HoldingModel|"
    r"DepositModel|StockPriceModel|OrderExecutionModel|InvestorFlowModel)"
    r"\.(select|get|get_by_id|get_or_none|get_or_create|create|"
    r"delete|drop_table|create_table|bulk_create|bulk_insert|insert_many)\("
)


def _py_files(directory: pathlib.Path) -> list[pathlib.Path]:
    return [p for p in directory.rglob("*.py") if "__pycache__" not in str(p)]


def _check_no_orm_queries(directory: pathlib.Path) -> list[str]:
    violations = []
    for path in _py_files(directory):
        source = path.read_text(encoding="utf-8")
        for match in _ORM_CLASS_QUERY_PATTERN.finditer(source):
            line_no = source[: match.start()].count("\n") + 1
            violations.append(
                f"{path.relative_to(ROOT.parent.parent)}:{line_no}: {match.group()}"
            )
    return violations


# --- GP-1: Repository layer owns all DB access ---


def test_web_has_no_direct_orm_queries() -> None:
    """web/ must not call ORM class-level query methods directly.

    All DB access goes through *Repository objects from ServiceContainer.
    FIX: Replace Model.select() with the appropriate repository method.
    REF: docs/architecture.md → "Data Access"
    """
    violations = _check_no_orm_queries(ROOT / "web")
    assert not violations, (
        "Direct ORM queries found in web/ (GP-1 violation):\n"
        + "\n".join(violations)
        + "\nFIX: Use container.*_repository methods instead of Model.select() / Model.get()."
    )


def test_services_has_no_direct_orm_queries() -> None:
    """services/ must not call ORM class-level query methods directly.

    All DB access goes through *Repository objects from ServiceContainer.
    FIX: Replace Model.select() with the appropriate repository method.
    REF: docs/architecture.md → "Data Access"
    """
    violations = _check_no_orm_queries(ROOT / "services")
    assert not violations, (
        "Direct ORM queries found in services/ (GP-1 violation):\n"
        + "\n".join(violations)
        + "\nFIX: Use *Repository methods injected via ServiceContainer."
    )


# --- GP-3: Layer dependency direction ---


def test_services_does_not_import_web() -> None:
    """services/ must not import from web/.

    Dependency direction: web/ → services/. Reverse is a cycle.
    FIX: Move shared logic to services/ or core/.
    REF: docs/architecture.md → "Layer Rules"
    """
    violations = []
    for path in _py_files(ROOT / "services"):
        source = path.read_text(encoding="utf-8")
        if "portfolio_manager.web" in source:
            violations.append(str(path.relative_to(ROOT.parent.parent)))
    assert not violations, "services/ imports web/ (GP-3 violation):\n" + "\n".join(
        violations
    )


def test_repositories_does_not_import_web() -> None:
    """repositories/ must not import from web/.

    Dependency direction: web/ → services/ → repositories/. Reverse is a cycle.
    NOTE: Importing services.database is allowed (contains Peewee ORM model classes).
    FIX: Move shared logic down to repositories/ or models/.
    REF: docs/architecture.md → "Layer Rules"
    """
    violations = []
    for path in _py_files(ROOT / "repositories"):
        source = path.read_text(encoding="utf-8")
        if "portfolio_manager.web" in source:
            violations.append(str(path.relative_to(ROOT.parent.parent)))
    assert not violations, "repositories/ imports web/ (GP-3 violation):\n" + "\n".join(
        violations
    )


def test_repositories_only_imports_database_from_services() -> None:
    """repositories/ may only import from services.database (Peewee models), not other services/.

    repositories/ must not depend on business logic services.
    FIX: Use dependency injection — pass service results as parameters, don't import the service.
    REF: docs/architecture.md → "Layer Rules"
    """
    violations = []
    for path in _py_files(ROOT / "repositories"):
        source = path.read_text(encoding="utf-8")
        # Strip allowed import: portfolio_manager.services.database
        cleaned = source.replace("portfolio_manager.services.database", "")
        if "portfolio_manager.services" in cleaned:
            violations.append(str(path.relative_to(ROOT.parent.parent)))
    assert not violations, (
        "repositories/ imports services other than services.database (GP-3 violation):\n"
        + "\n".join(violations)
        + "\nOnly services.database (Peewee ORM models) is allowed in repositories/."
    )


def test_models_does_not_import_other_layers() -> None:
    """models/ must not import from repositories/, services/, or web/.

    models/ is the bottom layer — it defines schema only.
    FIX: Remove cross-layer imports from model definitions.
    REF: docs/architecture.md → "Layer Rules"
    """
    violations = []
    for path in _py_files(ROOT / "models"):
        source = path.read_text(encoding="utf-8")
        for forbidden in (
            "portfolio_manager.repositories",
            "portfolio_manager.services",
            "portfolio_manager.web",
        ):
            if forbidden in source:
                violations.append(
                    f"{path.relative_to(ROOT.parent.parent)}: imports {forbidden}"
                )
    assert not violations, (
        "models/ imports upper layers (GP-3 violation):\n" + "\n".join(violations)
    )
