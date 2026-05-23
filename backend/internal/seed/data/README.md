# Seed data

Place reference data here. These will be loaded into the DB by a future
`seed` command.

Planned files:

- `countries.json`     — ISO 3166 country list with Bangla/English names.
- `bd_locations.json`  — Bangladesh divisions → districts → upazilas
                         (~8 / 64 / 495 entries) with bilingual names and
                         BBS codes. Sourced from public BBS / open datasets.
- `crime_types.json`   — Initial list of crime categories
                         (rape, assault, enforced disappearance, etc).

Until the seed command is implemented, the JSON files in this directory
are the source of truth and can be loaded manually into the database.
