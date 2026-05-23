// Package seed loads reference data (countries, BD locations, crime types)
// from JSON files embedded in the API binary into the database.
//
// All operations are idempotent: re-running the seed leaves the database in
// the same state. With --reset, previously seeded rows are removed first
// (only for the seed-owned tables; user data is never touched).
package seed

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed all:data
var seedFS embed.FS

// Run loads all seed datasets. If reset is true, the seed-owned tables are
// truncated first.
func Run(ctx context.Context, pool *pgxpool.Pool, reset bool, logger *slog.Logger) error {
	if reset {
		if err := truncate(ctx, pool, logger); err != nil {
			return fmt.Errorf("reset: %w", err)
		}
	}
	if err := seedCountries(ctx, pool, logger); err != nil {
		return fmt.Errorf("countries: %w", err)
	}
	if err := seedBDLocations(ctx, pool, logger); err != nil {
		return fmt.Errorf("bd locations: %w", err)
	}
	if err := seedCrimeTypes(ctx, pool, logger); err != nil {
		return fmt.Errorf("crime types: %w", err)
	}
	logger.Info("seed complete")
	return nil
}

func truncate(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	logger.Info("resetting seed tables")
	const stmt = `TRUNCATE upazilas, districts, divisions, countries, crime_types RESTART IDENTITY CASCADE`
	_, err := pool.Exec(ctx, stmt)
	return err
}

// ---------- Countries ----------

type countryRow struct {
	ISO2      string  `json:"iso2"`
	ISO3      string  `json:"iso3"`
	NameEN    string  `json:"name_en"`
	NameBN    *string `json:"name_bn"`
	PhoneCode *string `json:"phone_code"`
}

func seedCountries(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	raw, err := seedFS.ReadFile("data/countries.json")
	if err != nil {
		return fmt.Errorf("read countries.json: %w", err)
	}
	var rows []countryRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return fmt.Errorf("parse countries.json: %w", err)
	}

	const stmt = `
INSERT INTO countries (iso2, iso3, name_en, name_bn, phone_code)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (iso2) DO UPDATE
SET iso3       = EXCLUDED.iso3,
    name_en    = EXCLUDED.name_en,
    name_bn    = COALESCE(EXCLUDED.name_bn, countries.name_bn),
    phone_code = COALESCE(EXCLUDED.phone_code, countries.phone_code)
`
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)

	for _, r := range rows {
		if _, err := tx.Exec(ctx, stmt, r.ISO2, r.ISO3, r.NameEN, r.NameBN, r.PhoneCode); err != nil {
			return fmt.Errorf("upsert %s: %w", r.ISO2, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	logger.Info("seeded countries", "count", len(rows))
	return nil
}

// ---------- Bangladesh locations ----------

type bdSeed struct {
	Divisions []struct {
		ID     int    `json:"id"`
		NameEN string `json:"name_en"`
		NameBN string `json:"name_bn"`
	} `json:"divisions"`
	Districts []struct {
		ID         int    `json:"id"`
		DivisionID int    `json:"division_id"`
		NameEN     string `json:"name_en"`
		NameBN     string `json:"name_bn"`
	} `json:"districts"`
	Upazilas []struct {
		ID         int    `json:"id"`
		DistrictID int    `json:"district_id"`
		NameEN     string `json:"name_en"`
		NameBN     string `json:"name_bn"`
	} `json:"upazilas"`
}

func seedBDLocations(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	raw, err := seedFS.ReadFile("data/bd_locations.json")
	if err != nil {
		return fmt.Errorf("read bd_locations.json: %w", err)
	}
	var data bdSeed
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("parse bd_locations.json: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)

	// Resolve Bangladesh's country_id (it must already exist from seedCountries).
	var bdCountryID int
	if err := tx.QueryRow(ctx, `SELECT id FROM countries WHERE iso2 = 'BD'`).Scan(&bdCountryID); err != nil {
		return fmt.Errorf("lookup BD country: %w", err)
	}

	// Divisions ----
	const divStmt = `
INSERT INTO divisions (id, country_id, name_en, name_bn)
VALUES ($1, $2, $3, $4)
ON CONFLICT (country_id, name_en) DO UPDATE
SET name_bn = EXCLUDED.name_bn
`
	for _, d := range data.Divisions {
		if _, err := tx.Exec(ctx, divStmt, d.ID, bdCountryID, d.NameEN, d.NameBN); err != nil {
			return fmt.Errorf("upsert division %s: %w", d.NameEN, err)
		}
	}

	// Districts ----
	const distStmt = `
INSERT INTO districts (id, division_id, name_en, name_bn)
VALUES ($1, $2, $3, $4)
ON CONFLICT (division_id, name_en) DO UPDATE
SET name_bn = EXCLUDED.name_bn
`
	for _, d := range data.Districts {
		if _, err := tx.Exec(ctx, distStmt, d.ID, d.DivisionID, d.NameEN, d.NameBN); err != nil {
			return fmt.Errorf("upsert district %s: %w", d.NameEN, err)
		}
	}

	// Upazilas ----
	const upzStmt = `
INSERT INTO upazilas (id, district_id, name_en, name_bn)
VALUES ($1, $2, $3, $4)
ON CONFLICT (district_id, name_en) DO UPDATE
SET name_bn = EXCLUDED.name_bn
`
	for _, u := range data.Upazilas {
		if _, err := tx.Exec(ctx, upzStmt, u.ID, u.DistrictID, u.NameEN, u.NameBN); err != nil {
			return fmt.Errorf("upsert upazila %s: %w", u.NameEN, err)
		}
	}

	// SERIAL sequences must be advanced past the highest seeded id so that
	// future inserts without an explicit id do not collide.
	for _, table := range []string{"divisions", "districts", "upazilas"} {
		stmt := fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', 'id'), GREATEST((SELECT MAX(id) FROM %s), 1))`,
			table, table,
		)
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("setval %s: %w", table, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	logger.Info("seeded BD locations",
		"divisions", len(data.Divisions),
		"districts", len(data.Districts),
		"upazilas", len(data.Upazilas),
	)
	return nil
}

// ---------- Crime types ----------

type crimeTypeRow struct {
	Slug     string `json:"slug"`
	NameBN   string `json:"name_bn"`
	NameEN   string `json:"name_en"`
	Severity int16  `json:"severity"`
}

func seedCrimeTypes(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	raw, err := seedFS.ReadFile("data/crime_types.json")
	if err != nil {
		return fmt.Errorf("read crime_types.json: %w", err)
	}
	var rows []crimeTypeRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return fmt.Errorf("parse crime_types.json: %w", err)
	}

	const stmt = `
INSERT INTO crime_types (slug, name_bn, name_en, severity, is_active)
VALUES ($1, $2, $3, $4, TRUE)
ON CONFLICT (slug) DO UPDATE
SET name_bn  = EXCLUDED.name_bn,
    name_en  = EXCLUDED.name_en,
    severity = EXCLUDED.severity
`
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)
	for _, r := range rows {
		if _, err := tx.Exec(ctx, stmt, r.Slug, r.NameBN, r.NameEN, r.Severity); err != nil {
			return fmt.Errorf("upsert %s: %w", r.Slug, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	logger.Info("seeded crime types", "count", len(rows))
	return nil
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
