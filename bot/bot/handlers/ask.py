import asyncio

from aiogram import Router, F
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton

from bot.api_client import ApiError
from bot.formatters import format_ask_answer
from bot.handlers.search import resolve_scope
from bot.keyboards.callbacks import NavCb

router = Router()
HISTORY_LIMIT = 20


@router.message(Command("ask"))
async def cmd_ask(message: Message, state: FSMContext, api, front_url):
    q = (message.text or "").partition(" ")[2].strip()
    if not q:
        await state.update_data(ask_mode=True, ask_history=[])
        await message.answer("Задайте вопрос. Выйти из режима вопросов: /search")
        return
    await handle_question(message, q, state, api, front_url)


async def _typing(message: Message):
    while True:
        await message.bot.send_chat_action(message.chat.id, "typing")
        await asyncio.sleep(4)


async def handle_question(message: Message, q: str, state: FSMContext, api, front_url):
    if not (2 <= len(q) <= 1000):
        await message.answer("Вопрос должен быть от 2 до 1000 символов.")
        return
    await state.update_data(ask_mode=True)
    thinking = await message.answer("Думаю…")
    typing = asyncio.create_task(_typing(message))

    d = await state.get_data()
    scope = {k: v for k, v in (await resolve_scope(state)).items() if v}
    history = d.get("ask_history") or []
    try:
        resp = await api.ask(q, history=history, **scope)
    except ApiError as e:
        await thinking.edit_text(e.message)
        return
    finally:
        typing.cancel()

    kb = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="Завершить", callback_data=NavCb(action="end_ask").pack())],
    ])
    await thinking.edit_text(format_ask_answer(resp, front_url), reply_markup=kb,
                             disable_web_page_preview=True)

    history = history + [
        {"role": "user", "content": q},
        {"role": "assistant", "content": resp.get("answer", "")},
    ]
    await state.update_data(ask_history=history[-HISTORY_LIMIT:])


@router.callback_query(NavCb.filter(F.action == "end_ask"))
async def end_ask(cb: CallbackQuery, state: FSMContext):
    await state.update_data(ask_mode=False, ask_history=[])
    await cb.message.edit_reply_markup(reply_markup=None)
    await cb.message.answer("Режим вопросов завершён. Пишите запрос - буду искать.")
    await cb.answer()
