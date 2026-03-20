# DutyRound

A duty and occurrence management platform for teams. DutyRound helps organizations schedule recurring duties, track participant assignments, manage out-of-office periods, and maintain fair workload distribution through leaderboards and calendar views.

## Features

- **Occurrence scheduling** -- create and manage duty slots with configurable participant limits, dates, and group assignments
- **Self-service sign-up** -- participants can sign up for or withdraw from open occurrences
- **Calendar view** -- visual calendar with day-level drill-down to see scheduled occurrences
- **Leaderboard** -- track participation counts to ensure fair distribution across team members
- **Groups** -- organize occurrences by team or category
- **Out-of-office management** -- users declare unavailable periods; the system prevents conflicting assignments
- **User administration** -- admins create accounts, reset passwords, and assign roles
- **Search** -- find occurrences and users quickly
- **Profile pages** -- view personal participation history and manage account settings
- **Email notifications** -- automated digest emails for new occurrences and unfilled spots, with configurable SMTP and daily limits
- **Responsive design** -- mobile-friendly layout with collapsible sidebar and adaptive grids

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.25 |
| Web framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | SQLite (via [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)) |
| Sessions | [gin-contrib/sessions](https://github.com/gin-contrib/sessions) (cookie store) |
| Frontend | Server-rendered HTML templates, HTMX, vanilla CSS |
| Auth | bcrypt password hashing |

## Quick Start

### Prerequisites

- **Go 1.25+** (CGO must be enabled for SQLite)
- **GCC** (required by `mattn/go-sqlite3`)

### Build and Run

```bash
git clone https://github.com/robinlant/occurance-management.git
cd occurance-management

go mod download
go build -o dutyround ./cmd/server
./dutyround
```

The server starts on `http://localhost:8080` by default.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `dutyround.db` | Path to the SQLite database file |
| `SESSION_SECRET` | `dev-secret-change-in-production` | Secret key for signing session cookies |
| `PORT` | `8080` | HTTP listen port |

## Docker Deployment

### Pull and Run

```bash
docker pull ghcr.io/robinlant/dutyround:latest

docker run -d \
  --name dutyround \
  -p 8080:8080 \
  -v dutyround-data:/data \
  -e SESSION_SECRET="your-secure-secret-here" \
  ghcr.io/robinlant/dutyround:latest
```

The container stores the SQLite database at `/data/dutyround.db` by default. Mount a volume to `/data` to persist data across container restarts.

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
      DB_PATH: /data/dutyround.db
      PORT: "8080"
    restart: unless-stopped

volumes:
  dutyround-data:
```

### Build Locally

```bash
docker build -t dutyround .
docker run -d -p 8080:8080 -v dutyround-data:/data dutyround
```

## Administration

### Initial Setup

DutyRound does not ship with a default admin account. On first run the database tables are created automatically but contain no users. You need to insert the initial admin user directly into SQLite:

```bash
# Generate a bcrypt hash for your chosen password
# (use any bcrypt tool, or a short Go script)

sqlite3 dutyround.db "INSERT INTO users (name, email, role, password_hash) \
  VALUES ('Admin', 'admin@example.com', 'admin', '\$2a\$10\$YOUR_BCRYPT_HASH');"
```

Once logged in as admin, you can create all subsequent users through the web UI at `/users`.

### User Roles

| Role | Capabilities |
|------|-------------|
| **admin** | Full access. Create and delete users, reset passwords, manage groups, create/edit/delete occurrences, assign and remove participants. |
| **organizer** | Create, edit, and delete occurrences. Assign and remove participants. Manage groups. Cannot manage users. |
| **participant** | View occurrences and calendar. Sign up for and withdraw from open slots. Manage personal profile and out-of-office periods. View leaderboard. |

### Groups Management

Admins and organizers can create groups at `/groups` to categorize occurrences (e.g., by team, duty type, or location). Each occurrence can be assigned to one group. Groups can be used to filter the occurrences list and calendar views.

### Out-of-Office

Users manage their own out-of-office periods from the profile page (`/profile`). The system prevents adding out-of-office ranges that overlap with existing participation assignments, ensuring schedule consistency.

## Architecture Overview

DutyRound follows a layered architecture:

```
cmd/server/          -- Application entry point, router setup, migrations
internal/
  domain/            -- Core data models (User, Group, Occurrence, Participation, OutOfOffice)
  repository/        -- Repository interfaces
  repository/sqlite/ -- SQLite implementations
  service/           -- Business logic (UserService, GroupService, OccurrenceService)
  handler/           -- HTTP handlers, middleware, template rendering
  templates/         -- HTML templates (layouts, pages, partials)
migrations/          -- SQL migration files
static/              -- CSS and other static assets
```

**Request flow:** HTTP request --> Gin router --> Security headers --> CSRF middleware --> Auth middleware --> Handler --> Service --> Repository --> SQLite

The application uses server-side HTML rendering with Go's `html/template` package. Templates are parsed once and cached for performance. HTMX provides dynamic partial-page updates without a JavaScript framework. Sessions are stored in signed cookies with a 7-day expiry.

### Security

- **CSRF protection** -- per-session tokens validated on all POST requests (HTMX requests are exempt as they enforce same-origin)
- **Session security** -- `HttpOnly`, `SameSite=Lax`, and `Secure` (in production) cookie flags; session regeneration on password change
- **Security headers** -- `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`
- **Input validation** -- password minimum length enforcement, role validation against allowed values, email header injection prevention
- **Authorization** -- role-based access control at the route level; OOO deletion verified against record ownership
- **SQLite** -- WAL mode with busy timeout to prevent "database is locked" under concurrent access; atomic transactions for race-condition-prone operations (e.g. signup count check)

### Database Migrations

Migrations are tracked in a `schema_migrations` table. On startup the server reads `migrations/*.sql`, sorts alphabetically, and applies any that haven't been applied yet. To add a migration, create a new `.sql` file with the next sequence number (e.g. `004_my_change.sql`).

## Configuration Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_PATH` | No | `dutyround.db` | File path for the SQLite database |
| `SESSION_SECRET` | **Yes** (production) | `dev-secret-change-in-production` | Secret for cookie-based session signing. The server will **exit** if this is unset when `GIN_MODE=release`. |
| `PORT` | No | `8080` | TCP port the HTTP server listens on |
| `GIN_MODE` | No | `debug` | Set to `release` for production (enables secure cookies, enforces `SESSION_SECRET`) |
