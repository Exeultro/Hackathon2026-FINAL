package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"aba/internal/auth"
	"aba/internal/store"
)

type authAPI struct {
	store       store.Store
	secret      string
	botSecret   string
	botUsername string
}

type ctxKey string

const (
	ctxUserID     ctxKey = "user_id"
	ctxTelegramID ctxKey = "telegram_id"
)

const telegramCodeTTL = 10 * time.Minute

func (a *authAPI) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeValidation(w, map[string]any{"body": "invalid json"})
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	bad := map[string]any{}
	if a, err := mail.ParseAddress(in.Email); err != nil || a.Address != in.Email || len(in.Email) > 254 {
		bad["email"] = "Неверный email"
	}
	if n := utf8.RuneCountInString(in.Name); n == 0 || n > 64 {
		bad["name"] = "Имя от 1 до 64 символов"
	}
	if n := utf8.RuneCountInString(in.Password); n < 8 || n > 64 {
		bad["password"] = "Пароль от 8 до 64 символов"
	}
	if len(bad) > 0 {
		writeValidation(w, bad)
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeValidation(w, map[string]any{"password": err.Error()})
		return
	}
	u, err := a.store.CreateUser(r.Context(), store.NewID("usr_"), in.Email, in.Name, hash)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email_taken", "email already registered")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	a.issueTokens(w, r, u, http.StatusCreated)
}

func (a *authAPI) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeValidation(w, map[string]any{"body": "invalid json"})
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	u, hash, err := a.store.GetUserByEmail(r.Context(), in.Email)
	if err != nil || !auth.CheckPassword(hash, in.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	a.issueTokens(w, r, u, http.StatusOK)
}

func (a *authAPI) issueTokens(w http.ResponseWriter, r *http.Request, u *store.User, status int) {
	access, err := auth.GenerateAccessToken(a.secret, u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	refreshRaw, refreshHash := auth.GenerateRefreshToken()
	if err := a.store.SaveRefreshToken(r.Context(), u.ID, refreshHash, time.Now().Add(auth.RefreshTokenTTL)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, status, map[string]any{
		"access_token":  access,
		"refresh_token": refreshRaw,
		"token_type":    "Bearer",
		"expires_in":    int(auth.AccessTokenTTL.Seconds()),
		"user":          toAPIUser(*u),
	})
}

func (a *authAPI) refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.RefreshToken == "" {
		writeValidation(w, map[string]any{"refresh_token": "required"})
		return
	}
	userID, err := a.store.ConsumeRefreshToken(r.Context(), auth.HashRefreshToken(in.RefreshToken))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid refresh token")
		return
	}
	u, err := a.store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	a.issueTokens(w, r, u, http.StatusOK)
}

func (a *authAPI) logout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if in.RefreshToken != "" {
		_ = a.store.DeleteRefreshToken(r.Context(), auth.HashRefreshToken(in.RefreshToken))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *authAPI) me(w http.ResponseWriter, r *http.Request) {
	u, err := a.store.GetUserByID(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, toAPIUser(*u))
}

func toAPIUser(u store.User) map[string]any {
	var tg any
	if u.TelegramID != nil {
		tg = map[string]any{"id": *u.TelegramID, "username": u.TelegramUsername}
	}
	return map[string]any{
		"id": u.ID, "email": u.Email, "name": u.Name, "created_at": u.CreatedAt, "telegram": tg,
	}
}

// identify resolves the caller: a bearer token, or the Telegram bot (shared secret plus
// X-Telegram-Id) acting for a linked user. A broken credential is a 401 on any path.
func (a *authAPI) identify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if h := r.Header.Get("Authorization"); h != "" {
			claims, err := auth.ParseAccessToken(a.secret, strings.TrimPrefix(h, "Bearer "))
			if !strings.HasPrefix(h, "Bearer ") || err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			ctx = context.WithValue(ctx, ctxUserID, claims.UserID)
		} else if bs := r.Header.Get("X-Bot-Secret"); bs != "" {
			tgID, err := strconv.ParseInt(r.Header.Get("X-Telegram-Id"), 10, 64)
			if a.botSecret == "" || subtle.ConstantTimeCompare([]byte(bs), []byte(a.botSecret)) != 1 || err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "invalid bot credentials")
				return
			}
			ctx = context.WithValue(ctx, ctxTelegramID, tgID)
			u, err := a.store.GetUserByTelegram(ctx, tgID)
			if err != nil && !errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusInternalServerError, "internal", err.Error())
				return
			}
			if u != nil {
				ctx = context.WithValue(ctx, ctxUserID, u.ID)
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userID(r *http.Request) string {
	id, _ := r.Context().Value(ctxUserID).(string)
	return id
}

func (a *authAPI) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userID(r) == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *authAPI) telegramCode(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	code := hex.EncodeToString(b)
	if err := a.store.SaveTelegramCode(r.Context(), code, userID(r), time.Now().Add(telegramCodeTTL)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"code":       code,
		"bot_url":    "https://t.me/" + a.botUsername + "?start=link_" + code,
		"expires_in": int(telegramCodeTTL.Seconds()),
	})
}

func (a *authAPI) telegramLink(w http.ResponseWriter, r *http.Request) {
	tgID, ok := r.Context().Value(ctxTelegramID).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "bot credentials required")
		return
	}
	var in struct {
		Code     string `json:"code"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Code == "" {
		writeValidation(w, map[string]any{"code": "required"})
		return
	}
	anon := r.Header.Get("X-Client-Id")
	if store.IsUserOwner(anon) {
		anon = ""
	}
	u, err := a.store.LinkTelegram(r.Context(), in.Code, tgID, strings.TrimPrefix(in.Username, "@"), anon)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "invalid_code", "link code is invalid or expired")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toAPIUser(*u))
}

func (a *authAPI) telegramUnlink(w http.ResponseWriter, r *http.Request) {
	if err := a.store.UnlinkTelegram(r.Context(), userID(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
