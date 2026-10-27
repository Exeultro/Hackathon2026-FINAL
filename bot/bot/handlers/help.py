from aiogram import Router
from aiogram.filters import Command
from aiogram.types import Message

router = Router()


@router.message(Command("help"))
async def cmd_help(message: Message):
    await message.answer(
        "Просто напишите текст - найду фрагменты лекций.\n\n"
        "/search запрос — поиск\n"
        "/ask вопрос — вопрос по лекции/курсу\n"
        "/lectures — список лекций\n"
        "/courses — курсы\n"
        "/scope — текущая область поиска\n"
        "/add ссылка — добавить лекцию (YouTube, Vimeo, mp4)\n"
        "/bookmarks — закладки\n"
        "/account — привязка к аккаунту сайта (общие закладки)"
    )
