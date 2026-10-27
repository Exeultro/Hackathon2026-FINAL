from aiogram.filters.callback_data import CallbackData


class LecCb(CallbackData, prefix="lec"):
    action: str            # open, search, ask, summary, toc, terms
    lecture_id: str


class LecPageCb(CallbackData, prefix="lp"):
    page: int


class CourseCb(CallbackData, prefix="crs"):
    course_id: str


class NavCb(CallbackData, prefix="nav"):
    action: str            # more, less, all, ask_this, bookmark, end_ask, lectures, courses, demo
    lecture_id: str = ""
    sec: int = 0
    page: int = 0


class BmCb(CallbackData, prefix="bm"):
    action: str            # del
    bookmark_id: str


class AccCb(CallbackData, prefix="acc"):
    action: str            # unlink


class RetryCb(CallbackData, prefix="retry"):
    lecture_id: str
