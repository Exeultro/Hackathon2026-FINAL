from aiogram import Router, F
from aiogram.filters import CommandStart, CommandObject
from aiogram.fsm.context import FSMContext
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton

from bot.api_client import ApiError
from bot.keyboards.callbacks import NavCb

router = Router()
DEMO_QUERY = "теорема Пифагора"


@router.message(CommandStart())
async def cmd_start(message: Message, command: CommandObject, state: FSMContext, api, front_url):
    await state.clear()
    # диплинк /start lec_<id>
    payload = command.args or ""
    if payload.startswith("link_"):
        from bot.handlers.account import link_account
        await link_account(message, payload.removeprefix("link_"), api)
        return
    if payload.startswith("lec_"):
        from bot.handlers.lectures import send_lecture_card
        try:
            await send_lecture_card(message, payload, state, api, front_url)
            return
        except ApiError:
            pass

    kb = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text=f"🔍 Пример: {DEMO_QUERY}",
                              callback_data=NavCb(action="demo").pack())],
        [InlineKeyboardButton(text="Лекции", callback_data=NavCb(action="lectures").pack()),
         InlineKeyboardButton(text="Курсы", callback_data=NavCb(action="courses").pack())],
    ])
    await message.answer(
        "Привет! Я ищу по видеолекциям и отвечаю на вопросы.\n"
        "Напишите запрос своими словами - найду момент с таймкодом.\n"
        "Пришлите ссылку на видео - добавлю лекцию.\n"
        "Все команды: /help",
        reply_markup=kb,
    )


@router.callback_query(NavCb.filter(F.action == "demo"))
async def demo(cb: CallbackQuery, state: FSMContext, api, front_url):
    from bot.handlers.search import do_search
    await cb.answer()
    await do_search(cb.message, DEMO_QUERY, state, api, front_url)
