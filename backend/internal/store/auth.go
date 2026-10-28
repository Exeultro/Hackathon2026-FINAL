package store

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *pgStore) CreateUser(ctx context.Context, id, email, name, passwordHash string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (id, email, name, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, name, created_at, telegram_id, telegram_username`,
		id, email, name, passwordHash,
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &u.TelegramID, &u.TelegramUsername)
	if err != nil {
		if pgErr, ok := err.(interface{ SQLState() string }); ok && pgErr.SQLState() == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return &u, nil
}

func (s *pgStore) GetUserByEmail(ctx context.Context, email string) (*User, string, error) {
	var u User
	var hash string
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, name, created_at, telegram_id, telegram_username, password_hash FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &u.TelegramID, &u.TelegramUsername, &hash)
	if err == pgx.ErrNoRows {
		return nil, "", ErrNotFound
	}
	return &u, hash, err
}

const userCols = `id, email, name, created_at, telegram_id, telegram_username`

func (s *pgStore) getUser(ctx context.Context, where string, arg any) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE `+where+` = $1`, arg).
		Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &u.TelegramID, &u.TelegramUsername)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *pgStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.getUser(ctx, "id", id)
}

func (s *pgStore) GetUserByTelegram(ctx context.Context, telegramID int64) (*User, error) {
	return s.getUser(ctx, "telegram_id", telegramID)
}

func (s *pgStore) SaveTelegramCode(ctx context.Context, code, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO telegram_link_codes (code, user_id, expires_at) VALUES ($1, $2, $3)`,
		code, userID, expiresAt)
	return err
}

// LinkTelegram consumes a one-time code, moves the Telegram account to its user
// (unlinking it from anyone else) and hands the bot's anonymous bookmarks, lectures and courses over to the user.
func (s *pgStore) LinkTelegram(ctx context.Context, code string, telegramID int64, username, anonOwner string) (*User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var userID string
	err = tx.QueryRow(ctx, `
		DELETE FROM telegram_link_codes WHERE code = $1 AND expires_at > now()
		RETURNING user_id`, code).Scan(&userID)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var name *string
	if username != "" {
		name = &username
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET telegram_id = NULL, telegram_username = NULL
		WHERE telegram_id = $1 AND id <> $2`, telegramID, userID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET telegram_id = $2, telegram_username = $3 WHERE id = $1`,
		userID, telegramID, name); err != nil {
		return nil, err
	}
	if anonOwner != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE bookmarks SET client_id = $2, user_id = $3
			WHERE client_id = $1`, anonOwner, UserOwner(userID), userID); err != nil {
			return nil, err
		}
		for _, table := range []string{"lectures", "courses"} {
			if _, err := tx.Exec(ctx, `UPDATE `+table+` SET owner = $2 WHERE owner = $1`, anonOwner, UserOwner(userID)); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, userID)
}

func (s *pgStore) UnlinkTelegram(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET telegram_id = NULL, telegram_username = NULL WHERE id = $1`, userID)
	return err
}

const userOwnerPrefix = "user:"

// UserOwner is the owner key of a signed-in user's bookmarks and progress;
// anonymous rows are keyed by the raw X-Client-Id.
func UserOwner(userID string) string { return userOwnerPrefix + userID }

func ownerUserID(owner string) *string {
	if id, ok := strings.CutPrefix(owner, userOwnerPrefix); ok {
		return &id
	}
	return nil
}

func IsUserOwner(owner string) bool { return strings.HasPrefix(owner, userOwnerPrefix) }

func (s *pgStore) SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt)
	return err
}

func (s *pgStore) ConsumeRefreshToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx, `
		DELETE FROM refresh_tokens WHERE token_hash = $1 AND expires_at > now()
		RETURNING user_id`, tokenHash,
	).Scan(&userID)
	if err == pgx.ErrNoRows {
		return "", ErrNotFound
	}
	return userID, err
}

func (s *pgStore) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}
