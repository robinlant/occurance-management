# DutyRound

A self-hosted duty management platform for teams. Members sign up for recurring duties, organizers schedule and manage slots, and the leaderboard tracks fair workload distribution.

**Features:** duty scheduling with participant limits · self-service sign-up and withdrawal · calendar and list views · leaderboard with date-range filtering · groups to categorize duties · out-of-office periods with conflict detection · role-based access (admin / organizer / participant) · profile pages with activity heatmap · automated email digests · English and German UI

---

## Hosting with Docker

### Run

```bash
docker run -d \
  --name dutyround \
  -p 8080:8080 \
  -v dutyround-data:/data \
  -e SESSION_SECRET="your-secure-secret-here" \
  -e GIN_MODE=release \
  ghcr.io/robinlant/dutyround:latest
```

The database is stored at `/data/dutyround.db` inside the container. Mount a named volume (or a host path) to `/data` to persist it across restarts.

### Docker Compose

```yaml
services:
  dutyround:
    image: ghcr.io/robinlant/dutyround:latest
    ports:
      - "8080:8080"
    volumes:
      - dutyround-data:/data
    environment:
      SESSION_SECRET: "your-secure-secret-here"
      GIN_MODE: release
    restart: unless-stopped

volumes:
  dutyround-data:
```

---

## First-time Setup: Creating an Admin Account

DutyRound ships with no default users. After starting the container, use the bundled `seed` utility to create the initial admin account:

```bash
docker exec dutyround ./seed \
  -name="Admin" \
  -email="admin@example.com" \
  -password="your-password"
```

Output on success:
```
Admin created: id=1  name=Admin  email=admin@example.com
```

If an account with that email already exists, seed exits without making changes:
```
Notice: Admin with email admin@example.com already exists. No changes made.
```

Once logged in as admin, create all further users through the web UI at `/users`.

### seed flags

| Flag | Default | Description |
|------|---------|-------------|
| `-name` | *(required)* | Display name for the admin user |
| `-email` | *(required)* | Login email |
| `-password` | *(required)* | Password (minimum 8 characters) |
| `-db` | `$DB_PATH` or `dutyround.db` | Path to the SQLite database file |

---

## Migrations

Migrations run **automatically on every startup** — no manual step needed. The server reads all `*.sql` files from the `migrations/` directory (sorted alphabetically), checks which ones have already been applied, and applies any new ones.

Applied migrations are tracked in a `schema_migrations` table inside the database.

### Adding a migration (when self-building)

Create a new `.sql` file in `migrations/` with the next sequence number:

```bash
# Current latest is 009_recurrence_id_index.sql, so:
touch migrations/010_my_change.sql
```

Write your SQL, then rebuild and restart. The migration will be picked up automatically.

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SESSION_SECRET` | **Yes** (production) | `dev-secret-change-in-production` | Secret key for signing session cookies. The server exits on startup if this is unset when `GIN_MODE=release`. |
| `GIN_MODE` | No | `debug` | Set to `release` for production. Enables `Secure` flag on cookies and enforces `SESSION_SECRET`. |
| `DB_PATH` | No | `dutyround.db` | Path to the SQLite database file. In the official image this defaults to `/data/dutyround.db`. |
| `PORT` | No | `8080` | TCP port the HTTP server listens on. |

---

## User Roles

| Role | What they can do |
|------|-----------------|
| **admin** | Everything. Manage users, reset passwords, manage groups, create/edit/delete duties, assign and remove participants. |
| **organizer** | Create, edit, and delete duties. Assign and remove participants. Manage groups. Cannot manage users. |
| **participant** | View duties and calendar. Sign up for and withdraw from open slots. Manage own profile and out-of-office periods. |

---

## Building from Source

Requires Go with CGO enabled and GCC (needed by the SQLite driver).

```bash
git clone https://github.com/robinlant/dutyround.git
cd dutyround

CGO_ENABLED=1 go build -o dutyround ./cmd/server
CGO_ENABLED=1 go build -o seed ./cmd/seed

./dutyround
```

To build the Docker image locally:

```bash
docker build -t dutyround .
docker run -d -p 8080:8080 -v dutyround-data:/data dutyround
```
