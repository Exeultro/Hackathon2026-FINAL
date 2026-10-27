import asyncio
import logging

from aiogram import Bot, Dispatcher
from aiogram.client.default import DefaultBotProperties
from aiogram.enums import ParseMode
from aiogram.fsm.storage.memory import MemoryStorage
from aiogram.types import BotCommand, ErrorEvent

from bot.api_client import BackendClient, current_user
from bot.config import load_config
from bot.handlers import start, search, ask, lectures, courses, scope, add, bookmarks, account, help as help_h

logging.basicConfig(level=logging.INFO)


async def main() -> None:
    cfg = load_config()
    bot = Bot(
        token=cfg.bot_token,
        default=DefaultBotProperties(parse_mode=ParseMode.HTML),
    )
    dp = Dispatcher(storage=MemoryStorage())

    # Общий httpx-клиент на всё время жизни бота
    api = BackendClient(base_url=cfg.api_url, bot_secret=cfg.bot_secret)
    dp["api"] = api
    dp["front_url"] = cfg.front_url

    # Лекции и курсы личные: API должен знать, от чьего имени запрос
    @dp.update.outer_middleware()
    async def as_user(handler, event, data):
        user = data.get("event_from_user")
        token = current_user.set(user.id if user else None)
        try:
            return await handler(event, data)
        finally:
            current_user.reset(token)

    # Роутеры: порядок важен — команды раньше текстового fallback
    dp.include_routers(
        start.router,
        help_h.router,
        lectures.router,
        courses.router,
        scope.router,
        bookmarks.router,
        account.router,
        add.router,
        ask.router,        # /ask и текст в режиме вопросов
        search.router,     # /search и свободный текст — последним
    )

    @dp.error()
    async def on_error(event: ErrorEvent):
        logging.exception("Handler error", exc_info=event.exception)
        try:
            if event.update.message:
                await event.update.message.answer(
                    "Что-то пошло не так. Попробуйте ещё раз."
                )
            elif event.update.callback_query:
                await event.update.callback_query.answer(
                    "Что-то пошло не так. Попробуйте ещё раз.", show_alert=True
                )
        except Exception:
            pass

    try:
        await bot.delete_webhook(drop_pending_updates=True)
        await bot.set_my_commands([
            BotCommand(command="search", description="Поиск по лекциям"),
            BotCommand(command="ask", description="Вопрос по лекции или курсу"),
            BotCommand(command="lectures", description="Список лекций"),
            BotCommand(command="courses", description="Курсы"),
            BotCommand(command="scope", description="Область поиска"),
            BotCommand(command="add", description="Добавить лекцию по ссылке"),
            BotCommand(command="bookmarks", description="Закладки"),
            BotCommand(command="account", description="Привязка к аккаунту сайта"),
            BotCommand(command="help", description="Справка"),
        ])
        await dp.start_polling(bot)
    finally:
        await api.close()


if __name__ == "__main__":
    asyncio.run(main())