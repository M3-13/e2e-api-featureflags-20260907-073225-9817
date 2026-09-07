# Feature-Flag-Service

Ein schlanker, produktionsnaher Feature-Flag-Service als REST-API in Go. Flags
werden in einem thread-sicheren In-Memory-Store (`sync.RWMutex`) verwaltet und
über Endpunkte zum Anlegen, Auflisten, Lesen, Ändern, Löschen und Evaluieren
bereitgestellt. Der Service nutzt ausschließlich die Go-Standardbibliothek
(`net/http`) und benötigt keine externe Datenbank.

## Tech Stack

- **Sprache:** Go (≥ 1.22)
- **Framework:** keines — `net/http` Standardbibliothek
- **Modul:** `featureflag`
- **Storage:** In-Memory mit `sync.RWMutex`
- **Tests:** `go test` / `net/http/httptest`

## Installation

Keine externen Abhängigkeiten — nur eine Go-Toolchain wird benötigt.

```sh
go build ./...
```

## Start (Dev)

```sh
go run .
```

Der Server lauscht danach auf Port `8080`. Ein einzelner
`RUN.json`-Eintrag deklariert genau diesen Start.

## Bauen (Production)

```sh
go build -o featureflag .
./featureflag
```

## Endpunkte

Fehlerantworten haben immer die Form `{"error":"..."}`.

| Methode | Pfad                       | Beschreibung                                       |
|---------|----------------------------|----------------------------------------------------|
| POST    | `/flags`                   | Legt ein Flag an (`201`), `400` ungültig, `409` vorhanden |
| GET     | `/flags`                   | Listet alle Flags sortiert nach `key` (`200`)       |
| GET     | `/flags/{key}`             | Liefert ein Flag (`200`) oder `404`                 |
| PUT     | `/flags/{key}`             | Ändert ein Flag (`200`), `400` ungültig, `404` unbekannt |
| DELETE  | `/flags/{key}`             | Löscht ein Flag (`204`) oder `404`                  |
| GET     | `/flags/{key}/evaluate`    | Deterministische Evaluation pro Nutzer (`200`)      |
| GET     | `/healthz`                 | Health-Check: `200 {"status":"ok"}`                 |

### Request-/Response-Schema

**Flag:**

```json
{
  "key": "feature-x",
  "enabled": true,
  "description": "optional",
  "rollout_percent": 50
}
```

**POST /flags** — Body: `{key, enabled, description?, rollout_percent?}` → `201 Flag`.

**PUT /flags/{key}** — Body: `{enabled, description?, rollout_percent?}` → `200 Flag`.

**GET /flags/{key}/evaluate?user={id}** → `200 {key, user, enabled, result}`.

**GET /healthz** → `200 {"status":"ok"}`.

## Features

- Thread-sicherer In-Memory-Store als einzige Datenquelle
- CRUD-Endpunkte für Flags mit sauberer Eingabevalidierung
- Deterministische, nutzerbasierte Flag-Evaluation
- Health-Endpunkt `/healthz`
- Explizite Server-Timeouts (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`)
