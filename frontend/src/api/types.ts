import type { components } from './schema'

type Schemas = components['schemas']

export type Lecture = Schemas['Lecture']
export type LectureStatus = Schemas['LectureStatus']
export type SummaryStatus = Schemas['SummaryStatus']
export type Language = Schemas['Language']
export type Playback = Schemas['Playback']
export type Segment = Schemas['Segment']
export type Transcript = Schemas['Transcript']
export type OutlineItem = Schemas['OutlineItem']
export type KeyTerm = Schemas['KeyTerm']
export type Summary = Schemas['Summary']
export type SearchHit = Schemas['SearchHit']
export type SearchResponse = Schemas['SearchResponse']
export type ChatMessage = Schemas['ChatMessage']
export type AskRequest = Schemas['AskRequest']
export type AskResponse = Schemas['AskResponse']
export type Course = Schemas['Course']
export type SystemInfo = Schemas['SystemInfo']
export type Bookmark = Schemas['Bookmark']
export type BookmarkCreate = Schemas['BookmarkCreate']
export type WatchProgress = Schemas['WatchProgress']
export type User = Schemas['User']
export type AuthResponse = Schemas['AuthResponse']
export type RegisterRequest = Schemas['RegisterRequest']
export type LoginRequest = Schemas['LoginRequest']
export type TelegramCode = Schemas['TelegramCode']

export interface LectureInput {
  title?: string
  course_id?: string
  language?: Language
}

export interface SearchQuery {
  q: string
  lecture_id?: string
  course_id?: string
  limit?: number
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public details?: Record<string, unknown>,
  ) {
    super(message)
  }
}

export interface Api {
  listLectures(query?: { course_id?: string }): Promise<Lecture[]>
  getLecture(id: string): Promise<Lecture>
  uploadLecture(file: File, input: LectureInput, onProgress?: (share: number) => void): Promise<Lecture>
  importLecture(url: string, input: LectureInput): Promise<Lecture>
  updateLecture(id: string, patch: { title?: string; course_id?: string | null }): Promise<Lecture>
  deleteLecture(id: string): Promise<void>
  retryLecture(id: string): Promise<Lecture>
  getTranscript(id: string): Promise<Transcript>
  subtitlesUrl(id: string): Promise<string>
  getSummary(id: string): Promise<Summary>
  search(query: SearchQuery): Promise<SearchResponse>
  ask(req: AskRequest): Promise<AskResponse>
  listCourses(): Promise<Course[]>
  createCourse(title: string): Promise<Course>
  updateCourse(id: string, title: string): Promise<Course>
  deleteCourse(id: string): Promise<void>
  system(): Promise<SystemInfo>
  listBookmarks(lectureId?: string): Promise<Bookmark[]>
  createBookmark(body: BookmarkCreate): Promise<Bookmark>
  updateBookmark(id: string, patch: { note?: string; t?: number }): Promise<Bookmark>
  deleteBookmark(id: string): Promise<void>
  listProgress(): Promise<WatchProgress[]>
  saveProgress(lectureId: string, position: number): Promise<WatchProgress>
  clearProgress(): Promise<void>
  register(body: RegisterRequest): Promise<AuthResponse>
  login(body: LoginRequest): Promise<AuthResponse>
  refresh(refreshToken: string): Promise<AuthResponse>
  logout(refreshToken: string): Promise<void>
  me(): Promise<User>
  telegramCode(): Promise<TelegramCode>
  unlinkTelegram(): Promise<void>
}
