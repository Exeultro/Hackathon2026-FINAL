import html

from aiogram import Router, F
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton, BufferedInputFile

from bot.api_client import ApiError
from bot.formatters import fmt_timecode, front_link, md_to_html
from bot.keyboards.callbacks import LecCb, LecPageCb, NavCb

router = Router()
PAGE = 8


async def lectures_view(api, state: FSMContext, page: int) -> tuple[str, InlineKeyboardMarkup | None]:
    d = await state.get_data()
    course_id = d.get("scope_course_id") if d.get("scope_type") == "course" else None
    items = await api.list_lectures(course_id=course_id)
    if not items:
        return "Готовых лекций пока нет. Пришлите ссылку на видео - добавлю.", None
    pages = (len(items) + PAGE - 1) // PAGE
    page = max(0, min(page, pages - 1))
    rows = [
        [InlineKeyboardButton(text=it["title"][:60],
                              callback_data=LecCb(action="open", lecture_id=it["id"]).pack())]
        for it in items[page * PAGE:(page + 1) * PAGE]
    ]
    nav = []
    if page > 0:
        nav.append(InlineKeyboardButton(text="←", callback_data=LecPageCb(page=page - 1).pack()))
    if page < pages - 1:
        nav.append(InlineKeyboardButton(text="→", callback_data=LecPageCb(page=page + 1).pack()))
    if nav:
        rows.append(nav)
    title = "Лекции курса:" if course_id else "Лекции:"
    if pages > 1:
        title += f" стр. {page + 1}/{pages}"
    return title, InlineKeyboardMarkup(inline_keyboard=rows)


@router.message(Command("lectures"))
async def cmd_lectures(message: Message, state: FSMContext, api):
    try:
        text, kb = await lectures_view(api, state, 0)
    except ApiError as e:
        await message.answer(e.message)
        return
    await message.answer(text, reply_markup=kb)


@router.callback_query(NavCb.filter(F.action == "lectures"))
async def nav_lectures(cb: CallbackQuery, state: FSMContext, api):
    try:
        text, kb = await lectures_view(api, state, 0)
    except ApiError as e:
        await cb.answer(e.message, show_alert=True)
        return
    await cb.message.answer(text, reply_markup=kb)
    await cb.answer()


@router.callback_query(LecPageCb.filter())
async def lectures_page(cb: CallbackQuery, callback_data: LecPageCb, state: FSMContext, api):
    try:
        text, kb = await lectures_view(api, state, callback_data.page)
    except ApiError as e:
        await cb.answer(e.message, show_alert=True)
        return
    await cb.message.edit_text(text, reply_markup=kb)
    await cb.answer()


def lecture_card_kb(lid: str) -> InlineKeyboardMarkup:
    return InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="Искать в лекции",
                              callback_data=LecCb(action="search", lecture_id=lid).pack()),
         InlineKeyboardButton(text="Спросить",
                              callback_data=LecCb(action="ask", lecture_id=lid).pack())],
        [InlineKeyboardButton(text="Конспект",
                              callback_data=LecCb(action="summary", lecture_id=lid).pack()),
         InlineKeyboardButton(text="Оглавление",
                              callback_data=LecCb(action="toc", lecture_id=lid).pack()),
         InlineKeyboardButton(text="Термины",
                              callback_data=LecCb(action="terms", lecture_id=lid).pack())],
    ])


async def send_lecture_card(message: Message, lid: str, state: FSMContext, api, front_url):
    lec = await api.get_lecture(lid)
    await state.update_data(scope_type="lecture", scope_lecture_id=lec["id"], scope_course_id=None,
                            ask_history=[])
    dur = fmt_timecode(lec["duration"]) if lec.get("duration") else "—"
    caption = (f"<b>{html.escape(lec['title'])}</b>\nДлительность: {dur}\n"
               f'<a href="{front_link(front_url, lec["id"], 0)}">Открыть на сайте</a>\n\n'
               "Теперь поиск и вопросы идут по этой лекции. Сбросить: /scope")
    kb = lecture_card_kb(lec["id"])
    if lec.get("thumbnail_url"):
        try:
            photo = BufferedInputFile(await api.get_thumbnail(lec["id"]), filename="thumb.jpg")
            await message.answer_photo(photo, caption=caption, reply_markup=kb)
            return
        except ApiError:
            pass
    await message.answer(caption, reply_markup=kb, disable_web_page_preview=True)


@router.callback_query(LecCb.filter(F.action == "open"))
async def open_lecture(cb: CallbackQuery, callback_data: LecCb, state: FSMContext, api, front_url):
    try:
        await send_lecture_card(cb.message, callback_data.lecture_id, state, api, front_url)
    except ApiError as e:
        await cb.answer(e.message, show_alert=True)
        return
    await cb.answer()


@router.callback_query(LecCb.filter(F.action == "search"))
async def search_in_lecture(cb: CallbackQuery, callback_data: LecCb, state: FSMContext):
    await state.update_data(scope_type="lecture", scope_lecture_id=callback_data.lecture_id,
                            ask_mode=False)
    await cb.message.answer("Напишите, что найти в этой лекции.")
    await cb.answer()


@router.callback_query(LecCb.filter(F.action == "ask"))
async def ask_lecture(cb: CallbackQuery, callback_data: LecCb, state: FSMContext):
    await state.update_data(scope_type="lecture", scope_lecture_id=callback_data.lecture_id,
                            ask_mode=True, ask_history=[])
    await cb.message.answer("Задайте вопрос по лекции. Выйти из режима вопросов: /search")
    await cb.answer()


@router.callback_query(LecCb.filter(F.action.in_({"summary", "toc", "terms"})))
async def show_summary(cb: CallbackQuery, callback_data: LecCb, api, front_url):
    lid = callback_data.lecture_id
    try:
        data = await api.get_summary(lid)
    except ApiError as e:
        await cb.answer("Конспект ещё готовится" if e.code == "not_ready" else e.message, show_alert=True)
        return

    action = callback_data.action
    if action == "summary":
        for chunk in split_4096(md_to_html(data.get("summary") or "Конспект пуст.")):
            await cb.message.answer(chunk)
    elif action == "toc":
        for chunk in split_4096("<b>Оглавление</b>\n" + format_toc(data.get("outline") or [], front_url, lid)):
            await cb.message.answer(chunk, disable_web_page_preview=True)
    else:
        for chunk in split_4096("<b>Термины</b>\n" + format_terms(data.get("key_terms") or [], front_url, lid)):
            await cb.message.answer(chunk, disable_web_page_preview=True)
    await cb.answer()


def split_4096(text: str, limit: int = 4000) -> list[str]:
    out, buf = [], ""
    for para in text.split("\n"):
        while len(para) > limit:
            if buf:
                out.append(buf)
                buf = ""
            out.append(para[:limit])
            para = para[limit:]
        if buf and len(buf) + len(para) + 1 > limit:
            out.append(buf)
            buf = para
        else:
            buf = (buf + "\n" + para) if buf else para
    if buf.strip():
        out.append(buf)
    return out


def format_toc(items: list, front_url: str, lid: str, depth: int = 0) -> str:
    lines = []
    for it in items:
        url = front_link(front_url, lid, it.get("start", 0))
        indent = "    " * depth
        lines.append(f'{indent}• <a href="{url}">{fmt_timecode(it.get("start", 0))}</a> '
                     f'{html.escape(it.get("title", ""))}')
        if it.get("children"):
            lines.append(format_toc(it["children"], front_url, lid, depth + 1))
    return "\n".join(lines) or "Оглавление пусто."


def format_terms(items: list, front_url: str, lid: str) -> str:
    lines = []
    for t in items:
        line = f"<b>{html.escape(t.get('term', ''))}</b>"
        if t.get("definition"):
            line += f" — {html.escape(t['definition'])}"
        occ = t.get("occurrences") or []
        if occ:
            line += f' <a href="{front_link(front_url, lid, occ[0])}">[{fmt_timecode(occ[0])}]</a>'
        lines.append(line)
    return "\n".join(lines) or "Термины не найдены."
