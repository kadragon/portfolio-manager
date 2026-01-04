"""Shared behavior for KIS API clients."""


class KisBaseClient:
    """Base client with shared header and environment handling."""

    app_key: str
    app_secret: str
    access_token: str
    cust_type: str

    def _build_headers(self, tr_id: str) -> dict[str, str]:
        return {
            "content-type": "application/json",
            "authorization": f"Bearer {self.access_token}",
            "appkey": self.app_key,
            "appsecret": self.app_secret,
            "tr_id": tr_id,
            "custtype": self.cust_type,
        }

    @staticmethod
    def _tr_id_for_env(env: str, *, real_id: str, demo_id: str) -> str:
        env_normalized = env.strip().lower()
        if "/" in env_normalized:
            env_normalized = env_normalized.split("/", 1)[0]
        if env_normalized in {"real", "prod"}:
            return real_id
        if env_normalized in {"demo", "vps", "paper"}:
            return demo_id
        raise ValueError("env must be one of: real/prod or demo/vps/paper")
