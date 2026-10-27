import asyncio
import html
import logging

from aiogram import Router, F
from aiogram.filters import Command
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton

from bot.api_client import ApiError
from bot.keyboards.callbacks import LecCb, RetryCb

router = Router()
STAGES = {
    "queued": "в очереди",
    "downloading": "скачиваю видео",
    "extracting_audio": "извлекаю звук",
    "transcribing": "распознаю речь",
    "indexing": "строю поисковый индекс",
}
POLL_SEC = 3
POLL_LIMIT_SEC = 3 * 3600
_tasks: set[asyncio.Task] = set()


async def start_import(message: Message, url: str, api, front_url):
    try:
        lec = await api.import_lecture(url)
    except ApiError as e:
        await message.answer(e.message)
        return
    msg = await message.answer("⏳ Импорт: в очереди…")
    _spawn(msg, lec["id"], api)


@router.message(Command("add"))
async def cmd_add(message: Message, api, front_url):
    url = (message.text or "").partition(" ")[2].strip()
    if not url.startswith("http"):
        await message.answer("Пришлите ссылку: /add https://youtube.com/...")
        return
    await start_import(message, url, api, front_url)


def _spawn(msg: Message, lid: str, api):
    task = asyncio.create_task(_poll(msg, lid, api))
    _tasks.add(task)
    task.add_done_callback(_tasks.discard)


async def _poll(msg: Message, lid: str, api):
    shown = ""
    lec: dict = {}
    for _ in range(POLL_LIMIT_SEC // POLL_SEC):
        await asyncio.sleep(POLL_SEC)
        try:
            lec = await api.get_lecture(lid)
        except ApiError as e:
            if e.status == 404:
                await msg.edit_text("❌ Лекция удалена.")
                return
            continue
        st = lec.get("status")
        if st in ("ready", "failed"):
            break
        text = f"⏳ Импорт: {STAGES.get(st, st)}… {int(lec.get('progress', 0) * 100)}%"
        if lec.get("title") and lec["title"] != lec.get("source_url"):
            text = f"<b>{html.escape(lec['title'])}</b>\n" + text
        if text != shown:
            try:
                await msg.edit_text(text)
                shown = text
            except Exception:
                logging.debug("edit failed", exc_info=True)
    else:
        await msg.edit_text("⏳ Обработка идёт слишком долго, проверьте позже: /lectures")
        return

    title = html.escape(lec.get("title", "лекция"))
    if lec.get("status") == "ready":
        kb = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="Открыть карточку",
                                  callback_data=LecCb(action="open", lecture_id=lid).pack())],
        ])
        await msg.edit_text(f"✅ Готово: <b>{title}</b>\nКонспект появится чуть позже.", reply_markup=kb)
    else:
        err = html.escape(lec.get("error") or "неизвестная ошибка")
        kb = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="Повторить", callback_data=RetryCb(lecture_id=lid).pack())],
        ])
        await msg.edit_text(f"❌ Ошибка импорта: {err}", reply_markup=kb)


@router.callback_query(RetryCb.filter())
async def retry(cb: CallbackQuery, callback_data: RetryCb, api):
    try:
        await api.retry_import(callback_data.lecture_id)
    except ApiError as e:
        await cb.answer(e.message, show_alert=True)
        return
    await cb.message.edit_text("⏳ Импорт перезапущен…")
    _spawn(cb.message, callback_data.lecture_id, api)
    await cb.answer()
