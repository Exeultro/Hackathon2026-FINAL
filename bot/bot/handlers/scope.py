import html

from aiogram import Router, F
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton

from bot.api_client import ApiError

router = Router()


@router.message(Command("scope"))
async def cmd_scope(message: Message, state: FSMContext, api):
    d = await state.get_data()
    t = d.get("scope_type", "all")
    if t == "lecture":
        human = "одна лекция"
        try:
            human += f" «{html.escape((await api.get_lecture(d['scope_lecture_id']))['title'])}»"
        except ApiError:
            pass
    elif t == "course":
        human = "курс " + d.get("scope_title", "")
    else:
        human = "все лекции"
    kb = None
    if t != "all":
        kb = InlineKeyboardMarkup(inline_keyboard=[
            [InlineKeyboardButton(text="Сбросить (все лекции)", callback_data="scope_reset")],
        ])
    await message.answer(f"Текущая область: {human}", reply_markup=kb)


@router.callback_query(F.data == "scope_reset")
async def reset_scope(cb: CallbackQuery, state: FSMContext):
    await state.update_data(scope_type="all", scope_lecture_id=None, scope_course_id=None,
                            ask_history=[])
    await cb.message.edit_text("Область поиска: все лекции")
    await cb.answer()
