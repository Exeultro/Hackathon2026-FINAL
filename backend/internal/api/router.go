package api

import (
	"github.com/go-chi/chi/v5"

	"aba/internal/mlclient"
	"aba/internal/pipeline"
	"aba/internal/storage"
	"aba/internal/store"
)

func RegisterRoutes(r chi.Router, s store.Store, files *storage.Files, pipe *pipeline.Pipeline, ml *mlclient.Client, jwtSecret, botSecret, botUsername string) {
	a := &authAPI{store: s, secret: jwtSecret, botSecret: botSecret, botUsername: botUsername}
	mediaKey = []byte("media:" + jwtSecret)

	r.Get("/health", HealthHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", a.register)
		r.Post("/auth/login", a.login)
		r.Post("/auth/refresh", a.refresh)
		r.Post("/auth/logout", a.logout)

		r.Group(func(r chi.Router) {
			r.Use(a.identify)

			r.With(a.requireAuth).Get("/auth/me", a.me)
			r.With(a.requireAuth).Post("/auth/telegram/code", a.telegramCode)
			r.Post("/auth/telegram/link", a.telegramLink)
			r.With(a.requireAuth).Delete("/auth/telegram", a.telegramUnlink)

			r.Get("/lectures", ListLecturesHandler(s))
			r.Get("/lectures/{lecture_id}", GetLectureHandler(s))
			r.Post("/lectures/import", ImportLectureHandler(s, pipe))
			r.Post("/lectures/{lecture_id}/retry", RetryLectureHandler(s, pipe))
			r.Get("/lectures/{lecture_id}/video", GetLectureVideo(s, files))
			r.Head("/lectures/{lecture_id}/video", GetLectureVideo(s, files))
			r.Get("/lectures/{lecture_id}/transcript", GetTranscript(s))
			r.Get("/lectures/{lecture_id}/subtitles.vtt", GetSubtitles(s))
			r.Get("/lectures/{lecture_id}/summary", GetSummary(s))
			r.Get("/lectures/{lecture_id}/thumbnail", GetLectureThumbnail(s, files))
			r.Get("/search", Search(s, ml))
			r.Post("/ask", Ask(s, ml))
			r.Get("/courses", ListCoursesHandler(s))
			r.Get("/system", SystemHandler(ml))

			r.Get("/bookmarks", ListBookmarksHandler(s))
			r.Post("/bookmarks", CreateBookmarkHandler(s))
			r.Patch("/bookmarks/{bookmark_id}", UpdateBookmarkHandler(s))
			r.Delete("/bookmarks/{bookmark_id}", DeleteBookmarkHandler(s))
			r.Get("/progress", ListProgressHandler(s))
			r.Delete("/progress", ClearProgressHandler(s))
			r.Put("/lectures/{lecture_id}/progress", SaveProgressHandler(s))

			r.Group(func(r chi.Router) {
				r.Use(a.requireAuth)
				r.Post("/lectures", UploadLectureHandler(s, files, pipe))
				r.Patch("/lectures/{lecture_id}", UpdateLectureHandler(s))
				r.Delete("/lectures/{lecture_id}", DeleteLectureHandler(s, files))
				r.Post("/courses", CreateCourseHandler(s))
				r.Patch("/courses/{course_id}", UpdateCourseHandler(s))
				r.Delete("/courses/{course_id}", DeleteCourseHandler(s))
			})
		})
	})
}
