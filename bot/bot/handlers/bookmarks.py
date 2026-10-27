import html

from aiogram import Router, F
from aiogram.filters import Command
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton

from bot.api_client import ApiError
from bot.formatters import fmt_timecode, front_link
from bot.keyboards.callbacks import BmCb

router = Router()


@router.message(Command("bookmarks"))
async def cmd_bookmarks(message: Message, api, front_url):
    try:
        items = await api.list_bookmarks(message.from_user.id)
    except ApiError as e:
        await message.answer(e.message)
        return
    if not items:
        await message.answer("Закладок нет. Поставить можно кнопкой 🔖 под результатами поиска.")
        return
    for bm in items:
        url = front_link(front_url, bm["lecture_id"], bm.get("t", 0))
        text = f'<b>{html.escape(bm.get("lecture_title", "Лекция"))}</b> — <a href="{url}">{fmt_timecode(bm.get("t", 0))}</a>'
        if bm.get("note"):
            text += f"\n{html.escape(bm['note'])}"
        kb = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="Удалить",
                                  callback_data=BmCb(action="del", bookmark_id=bm["id"]).pack())]
        ])
        await message.answer(text, reply_markup=kb, disable_web_page_preview=True)


@router.callback_query(BmCb.filter(F.action == "del"))
async def delete_bm(cb: CallbackQuery, callback_data: BmCb, api):
    try:
        await api.delete_bookmark(cb.from_user.id, callback_data.bookmark_id)
    except ApiError as e:
        if e.status != 404:
            await cb.answer(e.message, show_alert=True)
            return
    await cb.message.delete()
    await cb.answer("Удалено")
