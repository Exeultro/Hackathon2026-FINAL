import uuid
from contextvars import ContextVar
from typing import Any

import httpx


class ApiError(Exception):
    def __init__(self, message: str, status: int | None = None, code: str | None = None):
        super().__init__(message)
        self.message = message
        self.status = status
        self.code = code


# Telegram user of the update being handled: every request goes out on their behalf
current_user: ContextVar[int | None] = ContextVar("current_user", default=None)


def client_id_for(user_id: int) -> str:
    return str(uuid.uuid5(uuid.NAMESPACE_URL, f"tg:{user_id}"))


class BackendClient:
    def __init__(self, base_url: str, bot_secret: str = ""):
        self._client = httpx.AsyncClient(base_url=base_url, timeout=15.0)
        self._bot_secret = bot_secret

    def _as(self, user_id: int) -> dict:
        # С секретом сервер узнаёт привязанный аккаунт по Telegram id, без него - аноним
        h = {"X-Client-Id": client_id_for(user_id)}
        if self._bot_secret:
            h["X-Bot-Secret"] = self._bot_secret
            h["X-Telegram-Id"] = str(user_id)
        return h

    async def close(self) -> None:
        await self._client.aclose()

    async def _send(self, method: str, url: str, *, timeout: float = 15.0, **kw) -> httpx.Response:
        uid = current_user.get()
        if uid is not None and "headers" not in kw:
            kw["headers"] = self._as(uid)
        try:
            r = await self._client.request(method, url, timeout=timeout, **kw)
        except httpx.TimeoutException as e:
            raise ApiError("Сервис долго не отвечает, попробуйте ещё раз") from e
        except httpx.RequestError as e:
            raise ApiError("Сервис недоступен, попробуйте через минуту") from e

        if r.status_code >= 400:
            code = None
            try:
                data = r.json()
                msg = data.get("message") or "Ошибка сервиса"
                code = data.get("code")
            except Exception:
                msg = "Ошибка сервиса"
            if r.status_code >= 500:
                msg = "Сервис недоступен, попробуйте через минуту"
            raise ApiError(msg, status=r.status_code, code=code)
        return r

    async def _request(self, method: str, url: str, **kw) -> Any:
        r = await self._send(method, url, **kw)
        if r.status_code == 204 or not r.content:
            return None
        return r.json()

    async def search(self, q: str, *, lecture_id: str | None = None,
                     course_id: str | None = None, limit: int = 10) -> dict:
        params = {"q": q, "limit": limit}
        if lecture_id:
            params["lecture_id"] = lecture_id
        if course_id:
            params["course_id"] = course_id
        return await self._request("GET", "/search", params=params)

    async def ask(self, question: str, *, lecture_id: str | None = None,
                  course_id: str | None = None, history: list | None = None) -> dict:
        body = {"question": question, "history": history or []}
        if lecture_id:
            body["lecture_id"] = lecture_id
        if course_id:
            body["course_id"] = course_id
        return await self._request("POST", "/ask", json=body, timeout=120.0)

    async def list_lectures(self, *, course_id: str | None = None) -> list:
        params = {"status": "ready"}
        if course_id:
            params["course_id"] = course_id
        return await self._request("GET", "/lectures", params=params)

    async def get_lecture(self, lecture_id: str) -> dict:
        return await self._request("GET", f"/lectures/{lecture_id}")

    async def get_thumbnail(self, lecture_id: str) -> bytes:
        r = await self._send("GET", f"/lectures/{lecture_id}/thumbnail")
        return r.content

    async def get_summary(self, lecture_id: str) -> dict:
        return await self._request("GET", f"/lectures/{lecture_id}/summary")

    async def list_courses(self) -> list:
        return await self._request("GET", "/courses")

    async def import_lecture(self, url: str) -> dict:
        return await self._request("POST", "/lectures/import", json={"url": url}, timeout=60.0)

    async def retry_import(self, lecture_id: str) -> dict:
        return await self._request("POST", f"/lectures/{lecture_id}/retry")

    async def list_bookmarks(self, user_id: int) -> list:
        return await self._request("GET", "/bookmarks", headers=self._as(user_id))

    async def add_bookmark(self, user_id: int, *, lecture_id: str, t: float, note: str = "") -> dict:
        return await self._request(
            "POST", "/bookmarks",
            headers=self._as(user_id),
            json={"lecture_id": lecture_id, "t": t, "note": note},
        )

    async def delete_bookmark(self, user_id: int, bookmark_id: str) -> None:
        await self._request(
            "DELETE", f"/bookmarks/{bookmark_id}",
            headers=self._as(user_id),
        )

    @property
    def can_link(self) -> bool:
        return bool(self._bot_secret)

    async def link_telegram(self, user_id: int, code: str, username: str | None) -> dict:
        return await self._request(
            "POST", "/auth/telegram/link",
            headers=self._as(user_id),
            json={"code": code, "username": username or ""},
        )

    async def me(self, user_id: int) -> dict | None:
        try:
            return await self._request("GET", "/auth/me", headers=self._as(user_id))
        except ApiError as e:
            if e.status == 401 and e.message == "missing token":
                return None
            raise

    async def unlink_telegram(self, user_id: int) -> None:
        await self._request("DELETE", "/auth/telegram", headers=self._as(user_id))
