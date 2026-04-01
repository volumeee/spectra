"""Spectra Python Client SDK"""
import requests
from typing import Any, Optional


class SpectraClient:
    def __init__(self, base_url: str, api_key: Optional[str] = None, timeout: int = 60):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout
        self.session = requests.Session()
        if api_key:
            self.session.headers["Authorization"] = f"Bearer {api_key}"

    def screenshot(self, url: str, **kwargs) -> Any:
        return self.execute("screenshot", "capture", {"url": url, **kwargs})

    def pdf(self, url: str, **kwargs) -> Any:
        return self.execute("pdf", "generate", {"url": url, **kwargs})

    def scrape(self, url: str, **kwargs) -> Any:
        return self.execute("scrape", "extract", {"url": url, **kwargs})

    def execute(self, plugin: str, method: str, params: dict) -> Any:
        resp = self.session.post(
            f"{self.base_url}/api/{plugin}/{method}",
            json=params,
            timeout=self.timeout,
        )
        data = resp.json()
        if not data.get("success"):
            error = data.get("error", {})
            raise Exception(f"[{error.get('code')}] {error.get('message')}")
        return data.get("data")

    def plugins(self) -> Any:
        resp = self.session.get(f"{self.base_url}/api/plugins", timeout=self.timeout)
        return resp.json()

    def health(self) -> bool:
        resp = self.session.get(f"{self.base_url}/health", timeout=self.timeout)
        return resp.ok
