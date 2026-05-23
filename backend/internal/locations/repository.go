// Package locations exposes read-only handlers for the
// country → division → district → upazila reference data.
package locations

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository encapsulates SQL access for location reference data.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Country is a flat row from the countries table.
type Country struct {
	ID        int     `json:"id"`
	ISO2      string  `json:"iso2"`
	ISO3      string  `json:"iso3"`
	NameEN    string  `json:"name_en"`
	NameBN    *string `json:"name_bn"`
	PhoneCode *string `json:"phone_code"`
}

// Division is a flat row from the divisions table.
type Division struct {
	ID        int     `json:"id"`
	CountryID int     `json:"country_id"`
	NameEN    string  `json:"name_en"`
	NameBN    *string `json:"name_bn"`
}

// District is a flat row from the districts table.
type District struct {
	ID         int     `json:"id"`
	DivisionID int     `json:"division_id"`
	NameEN     string  `json:"name_en"`
	NameBN     *string `json:"name_bn"`
}

// Upazila is a flat row from the upazilas table.
type Upazila struct {
	ID         int     `json:"id"`
	DistrictID int     `json:"district_id"`
	NameEN     string  `json:"name_en"`
	NameBN     *string `json:"name_bn"`
}

func (r *Repository) ListCountries(ctx context.Context) ([]Country, error) {
	const stmt = `
SELECT id, iso2, iso3, name_en, name_bn, phone_code
FROM   countries
WHERE  is_active = TRUE
ORDER BY name_en`
	rows, err := r.pool.Query(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Country, 0, 250)
	for rows.Next() {
		var c Country
		if err := rows.Scan(&c.ID, &c.ISO2, &c.ISO3, &c.NameEN, &c.NameBN, &c.PhoneCode); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) ListDivisions(ctx context.Context, countryID int) ([]Division, error) {
	const stmt = `
SELECT id, country_id, name_en, name_bn
FROM   divisions
WHERE  country_id = $1
ORDER BY name_en`
	rows, err := r.pool.Query(ctx, stmt, countryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Division, 0, 8)
	for rows.Next() {
		var d Division
		if err := rows.Scan(&d.ID, &d.CountryID, &d.NameEN, &d.NameBN); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) ListDistricts(ctx context.Context, divisionID int) ([]District, error) {
	const stmt = `
SELECT id, division_id, name_en, name_bn
FROM   districts
WHERE  division_id = $1
ORDER BY name_en`
	rows, err := r.pool.Query(ctx, stmt, divisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]District, 0, 16)
	for rows.Next() {
		var d District
		if err := rows.Scan(&d.ID, &d.DivisionID, &d.NameEN, &d.NameBN); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) ListUpazilas(ctx context.Context, districtID int) ([]Upazila, error) {
	const stmt = `
SELECT id, district_id, name_en, name_bn
FROM   upazilas
WHERE  district_id = $1
ORDER BY name_en`
	rows, err := r.pool.Query(ctx, stmt, districtID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Upazila, 0, 16)
	for rows.Next() {
		var u Upazila
		if err := rows.Scan(&u.ID, &u.DistrictID, &u.NameEN, &u.NameBN); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
