import html

from aiogram import Router, F
from aiogram.filters import Command
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton

from bot.api_client import ApiError
from bot.keyboards.callbacks import AccCb

router = Router()

NOT_CONFIGURED = "Привязка к сайту не настроена: боту и серверу нужен общий BOT_SECRET."


def how_to_link(front_url: str) -> str:
    return (
        "Telegram не привязан к аккаунту сайта.\n"
        f"Войдите на сайте, откройте Настройки ({front_url}/settings) и нажмите «Привязать Telegram» - "
        "после этого закладки здесь и на сайте станут общими."
    )


async def link_account(message: Message, code: str, api):
    if not api.can_link:
        await message.answer(NOT_CONFIGURED)
        return
    try:
        user = await api.link_telegram(message.from_user.id, code, message.from_user.username)
    except ApiError as e:
        if e.code == "invalid_code":
            await message.answer("Ссылка для привязки устарела или уже использована. Нажмите «Привязать Telegram» на сайте ещё раз.")
        else:
            await message.answer(e.message)
        return
    await message.answer(
        f"Готово: Telegram привязан к аккаунту <b>{html.escape(user['name'])}</b> ({html.escape(user['email'])}).\n"
        "Закладки здесь и на сайте теперь общие, ваши прежние закладки из бота перенесены в аккаунт.\n"
        "Отвязать: /account"
    )


@router.message(Command("account"))
async def cmd_account(message: Message, api, front_url):
    if not api.can_link:
        await message.answer(NOT_CONFIGURED)
        return
    try:
        user = await api.me(message.from_user.id)
    except ApiError as e:
        await message.answer(e.message)
        return
    if user is None:
        await message.answer(how_to_link(front_url))
        return
    kb = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="Отвязать", callback_data=AccCb(action="unlink").pack())]
    ])
    await message.answer(
        f"Telegram привязан к аккаунту <b>{html.escape(user['name'])}</b> ({html.escape(user['email'])}).\n"
        "Закладки здесь и на сайте общие.",
        reply_markup=kb,
    )


@router.callback_query(AccCb.filter(F.action == "unlink"))
async def unlink(cb: CallbackQuery, api):
    try:
        await api.unlink_telegram(cb.from_user.id)
    except ApiError as e:
        if e.status != 401:
            await cb.answer(e.message, show_alert=True)
            return
    await cb.message.edit_text("Telegram отвязан. Закладки остались в аккаунте на сайте, здесь начнётся новый список.")
    await cb.answer("Отвязано")
