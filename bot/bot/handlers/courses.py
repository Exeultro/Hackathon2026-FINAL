import html

from aiogram import Router, F
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.types import Message, CallbackQuery, InlineKeyboardMarkup, InlineKeyboardButton

from bot.api_client import ApiError
from bot.keyboards.callbacks import CourseCb, NavCb

router = Router()


async def courses_view(api) -> tuple[str, InlineKeyboardMarkup | None]:
    items = await api.list_courses()
    if not items:
        return "Курсов нет.", None
    kb = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text=f'{c["title"][:50]} ({c["lecture_count"]})',
                              callback_data=CourseCb(course_id=c["id"]).pack())]
        for c in items
    ])
    return "Выберите курс - он станет областью поиска:", kb


@router.message(Command("courses"))
async def cmd_courses(message: Message, api):
    try:
        text, kb = await courses_view(api)
    except ApiError as e:
        await message.answer(e.message)
        return
    await message.answer(text, reply_markup=kb)


@router.callback_query(NavCb.filter(F.action == "courses"))
async def nav_courses(cb: CallbackQuery, api):
    try:
        text, kb = await courses_view(api)
    except ApiError as e:
        await cb.answer(e.message, show_alert=True)
        return
    await cb.message.answer(text, reply_markup=kb)
    await cb.answer()


@router.callback_query(CourseCb.filter())
async def pick_course(cb: CallbackQuery, callback_data: CourseCb, state: FSMContext, api):
    title = "выбранный курс"
    try:
        for c in await api.list_courses():
            if c["id"] == callback_data.course_id:
                title = f"«{html.escape(c['title'])}»"
    except ApiError:
        pass
    await state.update_data(scope_type="course",
                            scope_course_id=callback_data.course_id,
                            scope_lecture_id=None,
                            scope_title=title,
                            ask_history=[])
    kb = InlineKeyboardMarkup(inline_keyboard=[
        [InlineKeyboardButton(text="Лекции курса", callback_data=NavCb(action="lectures").pack())],
    ])
    await cb.message.edit_text(f"Область поиска: курс {title} ✅\nПишите запрос.", reply_markup=kb)
    await cb.answer()
