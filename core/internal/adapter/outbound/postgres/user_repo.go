package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domainauth "github.com/radhakrishna/archbattle/core/internal/domain/auth"
	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, user *domainauth.User) error {
	role := user.Role
	if role == "" {
		role = domainauth.RoleUser
	}
	_, err := r.pool.Exec(ctx, `
        INSERT INTO users (id, username, email, password_hash, role, tier, junior_elo, senior_elo, staff_elo, matches_played, current_streak, longest_streak, last_daily_date, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
    `, user.ID, user.Username, strings.ToLower(user.Email), user.PasswordHash, role, user.Tier, user.JuniorELO, user.SeniorELO, user.StaffELO, user.MatchesPlayed, user.CurrentStreak, user.LongestStreak, user.LastDailyDate, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*domainauth.User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, username, email, password_hash, role, tier, junior_elo, senior_elo, staff_elo, matches_played, current_streak, longest_streak, last_daily_date, created_at
        FROM users WHERE username = $1
    `, username)
	return scanUser(row)
}

func (r *UserRepo) UpsertByUsername(ctx context.Context, username string) (*domainauth.User, error) {
	existing, err := r.FindByUsername(ctx, username)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("find user: %w", err)
	}
	user := &domainauth.User{
		ID:           uuid.New(),
		Username:     username,
		Email:        username + "@local",
		PasswordHash: "dummy",
		Role:         domainauth.RoleUser,
		Tier:         shared.TierJunior,
		JuniorELO:    1000,
		SeniorELO:    1000,
		StaffELO:     1000,
		CreatedAt:    time.Now().UTC(),
	}
	err = r.Create(ctx, user)
	if err != nil {
		existing, findErr := r.FindByUsername(ctx, username)
		if findErr != nil {
			return nil, fmt.Errorf("upsert user: %w", err)
		}
		return existing, nil
	}
	return user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domainauth.User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, username, email, password_hash, role, tier, junior_elo, senior_elo, staff_elo, matches_played, current_streak, longest_streak, last_daily_date, created_at
        FROM users WHERE email = $1
    `, strings.ToLower(email))
	return scanUser(row)
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainauth.User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, username, email, password_hash, role, tier, junior_elo, senior_elo, staff_elo, matches_played, current_streak, longest_streak, last_daily_date, created_at
        FROM users WHERE id = $1
    `, id)
	return scanUser(row)
}

func (r *UserRepo) UpdateELO(ctx context.Context, id uuid.UUID, tier shared.Tier, newELO int) error {
	column := eloColumn(tier)
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`UPDATE users SET %s = $2 WHERE id = $1`, column), id, newELO)
	if err != nil {
		return fmt.Errorf("update elo: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdateStreak(ctx context.Context, id uuid.UUID, streak int, lastDate time.Time) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE users
        SET current_streak = $2,
            longest_streak = GREATEST(longest_streak, $2),
            last_daily_date = $3
        WHERE id = $1
    `, id, streak, normalizeDate(lastDate))
	if err != nil {
		return fmt.Errorf("update streak: %w", err)
	}
	return nil
}

func (r *UserRepo) IncrementMatchesPlayed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET matches_played = matches_played + 1 WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("increment matches played: %w", err)
	}
	return nil
}

func (r *UserRepo) GetPlayerProfiles(ctx context.Context, tier shared.Tier, userIDs []uuid.UUID) ([]domainmatch.PlayerProfile, error) {
	column := eloColumn(tier)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
        SELECT id, username, %s, matches_played
        FROM users
        WHERE id = ANY($1)
    `, column), userIDs)
	if err != nil {
		return nil, fmt.Errorf("query player profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]domainmatch.PlayerProfile, 0, len(userIDs))
	for rows.Next() {
		var profile domainmatch.PlayerProfile
		if err := rows.Scan(&profile.UserID, &profile.Username, &profile.CurrentELO, &profile.MatchesPlayed); err != nil {
			return nil, fmt.Errorf("scan player profile: %w", err)
		}
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

func (r *UserRepo) UpdatePlayerProgress(ctx context.Context, tier shared.Tier, standing domainmatch.PlayerStanding) error {
	column := eloColumn(tier)
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`
        UPDATE users
        SET %s = $2,
            matches_played = $3
        WHERE id = $1
    `, column), standing.UserID, standing.ELOAfter, standing.MatchesPlayed)
	if err != nil {
		return fmt.Errorf("update player progress: %w", err)
	}
	return nil
}

func scanUser(scanner rowScanner) (*domainauth.User, error) {
	user := &domainauth.User{}
	if err := scanner.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Role, &user.Tier, &user.JuniorELO, &user.SeniorELO, &user.StaffELO, &user.MatchesPlayed, &user.CurrentStreak, &user.LongestStreak, &user.LastDailyDate, &user.CreatedAt); err != nil {
		return nil, err
	}
	return user, nil
}

func eloColumn(tier shared.Tier) string {
	switch tier {
	case shared.TierSenior:
		return "senior_elo"
	case shared.TierStaff:
		return "staff_elo"
	default:
		return "junior_elo"
	}
}
