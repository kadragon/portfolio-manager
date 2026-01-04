import os
import subprocess
import sys
from pathlib import Path


def test_overseas_price_script_runs():
    script = (
        Path(__file__).resolve().parents[3] / "scripts" / "check_kis_overseas_price.py"
    )
    env = {
        **os.environ,
        "KIS_OVERSEAS_EXCD": "NAS",
        "KIS_OVERSEAS_SYMB": "AAPL",
    }
    result = subprocess.run(
        [sys.executable, str(script)],
        env=env,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0
    assert "OK:" in result.stdout
