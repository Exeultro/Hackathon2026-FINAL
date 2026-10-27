import re

from aiogram import Router, F
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton

from bot.api_client import ApiError
from bot.formatters import format_search_results, playbacks_of
from bot.keyboards.callbacks import NavCb

router = Router()
URL_RE = re.compile(r"^https?://\S+$", re.I)
PAGE = 5


async def resolve_scope(state: FSMContext) -> dict:
    d = await state.get_data()
    return {
        "lecture_id": d.get("scope_lecture_id") if d.get("scope_type") == "lecture" else None,
        "course_id": d.get("scope_course_id") if d.get("scope_type") == "course" else None,
    }


def _results_kb(hits: list, shown: int, scoped: bool) -> InlineKeyboardMarkup:
    rows = []
    if not hits:
        if scoped:
            rows.append([InlineKeyboardButton(text="Искать по всем лекциям",
                                              callback_data=NavCb(action="all").pack())])
        return InlineKeyboardMarkup(inline_keyboard=rows)
    marks = [
        InlineKeyboardButton(
            text=f"🔖 {i}",
            callback_data=NavCb(action="bookmark", lecture_id=h["lecture"]["id"], sec=int(h["start"])).pack(),
        )
        for i, h in enumerate(hits[:shown], start=1)
    ]
    rows.append(marks)
    nav = []
    if len(hits) > shown:
        nav.append(InlineKeyboardButton(text="Ещё", callback_data=NavCb(action="more").pack()))
    elif shown > PAGE:
        nav.append(InlineKeyboardButton(text="Свернуть", callback_data=NavCb(action="less").pack()))
    nav.append(InlineKeyboardButton(text="Спросить об этом", callback_data=NavCb(action="ask_this").pack()))
    rows.append(nav)
    return InlineKeyboardMarkup(inline_keyboard=rows)


@router.message(Command("search"))
async def cmd_search(message: Message, state: FSMContext, api, front_url):
    q = (message.text or "").partition(" ")[2].strip()
    await state.update_data(ask_mode=False, ask_history=[])
    if not q:
        await message.answer("Напишите запрос: /search теорема Пифагора")
        return
    await do_search(message, q, state, api, front_url)


@router.message(F.text & ~F.text.startswith("/"))
async def free_text(message: Message, state: FSMContext, api, front_url):
    d = await state.get_data()
    text = (message.text or "").strip()

    if URL_RE.match(text):
        from bot.handlers.add import start_import
        await start_import(message, text, api, front_url)
        return

    if d.get("ask_mode"):
        from bot.handlers.ask import handle_question
        await handle_question(message, text, state, api, front_url)
        return

    await do_search(message, text, state, api, front_url)


async def do_search(message: Message, q: str, state: FSMContext, api, front_url, scope: dict | None = None):
    if not (2 <= len(q) <= 500):
        await message.answer("Запрос должен быть от 2 до 500 символов.")
        return
    if scope is None:
        scope = await resolve_scope(state)
    await message.bot.send_chat_action(message.chat.id, "typing")
    try:
        data = await api.search(q, limit=10, **scope)
        playbacks = playbacks_of(await api.list_lectures())
    except ApiError as e:
        await message.answer(e.message)
        return

    hits = data.get("hits") or []
    await state.update_data(last_search={"data": data, "playbacks": playbacks}, last_query=q)
    scoped = bool(scope.get("lecture_id") or scope.get("course_id"))
    await message.answer(
        format_search_results(data, front_url, PAGE, playbacks),
        reply_markup=_results_kb(hits, PAGE, scoped),
        disable_web_page_preview=True,
    )


async def _show_page(cb: CallbackQuery, state: FSMContext, front_url, shown: int):
    last = (await state.get_data()).get("last_search")
    if not last:
        await cb.answer("Результаты устарели, повторите запрос", show_alert=True)
        return
    hits = last["data"].get("hits") or []
    await cb.message.edit_text(
        format_search_results(last["data"], front_url, shown, last["playbacks"]),
        reply_markup=_results_kb(hits, shown, True),
        disable_web_page_preview=True,
    )
    await cb.answer()


@router.callback_query(NavCb.filter(F.action == "more"))
async def more(cb: CallbackQuery, state: FSMContext, front_url):
    await _show_page(cb, state, front_url, 2 * PAGE)


@router.callback_query(NavCb.filter(F.action == "less"))
async def less(cb: CallbackQuery, state: FSMContext, front_url):
    await _show_page(cb, state, front_url, PAGE)


@router.callback_query(NavCb.filter(F.action == "all"))
async def search_all(cb: CallbackQuery, state: FSMContext, api, front_url):
    q = (await state.get_data()).get("last_query")
    await cb.answer()
    if q:
        await do_search(cb.message, q, state, api, front_url, scope={})


@router.callback_query(NavCb.filter(F.action == "bookmark"))
async def bookmark(cb: CallbackQuery, callback_data: NavCb, api):
    try:
        await api.add_bookmark(cb.from_user.id, lecture_id=callback_data.lecture_id, t=callback_data.sec)
    except ApiError as e:
        await cb.answer(e.message, show_alert=True)
        return
    await cb.answer("Закладка сохранена: /bookmarks")


@router.callback_query(NavCb.filter(F.action == "ask_this"))
async def ask_this(cb: CallbackQuery, state: FSMContext):
    await state.update_data(ask_mode=True, ask_history=[])
    await cb.answer()
    await cb.message.answer("Задайте вопрос - отвечу по найденным лекциям. Выйти: /search")
