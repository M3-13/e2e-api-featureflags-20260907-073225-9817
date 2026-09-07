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

## Konfiguration

Der Service wird über Umgebungsvariablen konfiguriert. Alle sind optional,
fehlende Werte führen zu einem dokumentierten Fallback bzw. einer
abgeschalteten Funktion.

| Variable        | Bedeutung                                                        | Vorgabe / Fallback                     |
|-----------------|------------------------------------------------------------------|----------------------------------------|
| `AUTH_TOKEN`    | Bearer-Token für die API-Authentifizierung (`Authorization: Bearer …`) | Keine Authentifizierung (offener Zugriff) |
| `TLS_CERT_FILE` | Pfad zur TLS-Server-Zertifikatskette (PEM)                        | Kein TLS, unverschlüsseltes HTTP       |
| `TLS_KEY_FILE`  | Pfad zum privaten TLS-Schlüssel (PEM)                            | Kein TLS, unverschlüsseltes HTTP       |

`TLS_CERT_FILE` und `TLS_KEY_FILE` müssen gemeinsam gesetzt werden. Sind beide
gesetzt, lauscht der Server über TLS; andernfalls unverschlüsselt (siehe
„Betriebsvorgabe“).

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

## Datenschutz & Rechtsgrundlage

Der Service verarbeitet eine **User-ID ausschließlich transient** während der
Evaluierung (`GET /flags/{key}/evaluate?user={id}`): Sie wird in derselben
Anfrage gelesen, zusammen mit dem Flag-Schlüssel per FNV-1a gehasht und
unmittelbar verworfen. Es findet **keine persistente Speicherung** von
User-IDs statt; der In-Memory-Store enthält ausschließlich Flag-Daten
(`key`, `enabled`, `description`, `rollout_percent`).

Rechtsgrundlage der Verarbeitung ist **Art. 6 Abs. 1 lit. b DSGVO** (Erfüllung
eines Vertrags bzw. vorvertraglicher Maßnahmen), soweit die Evaluierung zur
Bereitstellung der vertraglich vereinbarten Funktion dient. Eine
interessensabwägung (lit. f) oder eine Einwilligung (lit. a) ist für den
transienten, nicht personenbezogen gespeicherten Evaluierungsschritt in der
Regel nicht erforderlich.

Der **Betreiber** ist verantwortlich dafür, diese Verarbeitung in seinem
Verarbeitungsverzeichnis zu dokumentieren und — soweit nach Art. 35 DSGVO
erforderlich — eine **Datenschutz-Folgenabschätzung (DSFA)** durchzuführen.
Die User-ID selbst sollte der Betreiber möglichst als pseudonymen oder
technischen Bezeichner ohne direkten Personenbezug wählen.

## Betrieb / Logging

Zugriffslogs werden über den Logger (`log` auf `stdout`) ausgegeben — eine
Zeile je Anfrage mit **Methode, Pfad (ohne Query-String), Status und Dauer**.
Der Query-String wird bewusst verworfen, damit Parameter wie `user` niemals in
den Logs erscheinen. Es werden **keine personenbezogenen Daten (PII)** geloggt.

Aufbewahrung, Rotation und Löschung der Logs liegen bei der Betriebsumgebung
(Systemd-Journal, Container-Runtime, Log-Aggregator) und nicht beim Service
selbst. Der Service schreibt ausschließlich nach `stdout`.

## Betriebsvorgabe

Der Service darf **nur hinter einer TLS-Terminierung** (Reverse-Proxy/Load
Balancer mit gültigem Zertifikat) **oder mit direktem TLS**
(`TLS_CERT_FILE`/`TLS_KEY_FILE`) betrieben werden. **Unverschlüsseltes
öffentliches HTTP ist nicht zulässig**, da über die API Flag-Konfiguration
geschrieben und `AUTH_TOKEN` übertragen wird. Ein rein unverschlüsselter
Betrieb ist ausschließlich für lokale Entwicklung vorgesehen.

## SBOM / Abhängigkeiten

Der Service nutzt **ausschließlich die Go-Standardbibliothek** (`net/http`,
`encoding/json`, `hash/fnv`, `sync`, `log`, …). Es bestehen **keine externen
Module** und damit keine Drittanbieter-Abhängigkeiten, die in einer SBOM
geführt oder auf bekannte Schwachstellen geprüft werden müssten. `go.mod`
enthält keine `require`-Einträge über die Standardbibliothek hinaus.

## Security Properties

- **Server-Timeouts:** `ReadHeaderTimeout` = 5 s, `ReadTimeout` = 5 s,
  `WriteTimeout` = 10 s, `IdleTimeout` = 60 s.
- **Body-Limit:** Anfrage-Bodies sind auf **1 MiB** begrenzt (`MaxBytesReader`);
  größere Anfragen werden mit `413` abgewiesen.
- **Content-Type-Pflicht:** Schreibende Endpunkte (`POST`/`PUT`) akzeptieren
  ausschließlich `Content-Type: application/json`, andernfalls `415`.
- **Generische Fehlertexte:** Fehlerantworten nutzen die Form `{"error":"…"}`
  und geben keine internen Details preis.
- **Logging ohne Query-String:** Zugriffslogs enthalten den Pfad ohne
  Query-Parameter (kein `user`-Leak).
- **In-Memory nur Flag-Daten:** Der Store persistiert ausschließlich
  Flag-Konfiguration; keine User-IDs oder Sessions.
- **Authentifizierung:** Optionale Absicherung über `AUTH_TOKEN`
  (Bearer-Token im `Authorization`-Header).
- **TLS:** Optionale direkte TLS-Bindung über `TLS_CERT_FILE`/`TLS_KEY_FILE`
  (siehe „Betriebsvorgabe“).
- **Rate-Limiting:** Anfragen werden je Client begrenzt (Rate-Limiting-Middleware).

## Wartung und Updates

- **Verantwortlichkeit:** Wartung, Updates und Sicherheits-Patches liegen beim
  Betreiber bzw. dem zuständigen Entwicklungsteam dieses Services.
- **Versionshistorie:** Änderungen werden über das Versionskontrollsystem
  (Git) nachvollzogen; Releases sind über Tags/Commits eindeutig referenzierbar.
- **Verfahren für Sicherheits-Patches:** Sicherheitslücken werden als
  prioritäre Änderung behandelt, im Repository behoben, durch `go build ./...`
  und `go test ./...` verifiziert und als Hotfix-Revision ausgeliefert.
  Da keine externen Module verwendet werden, entfällt das Nachziehen von
  Drittanbieter-Patches; Updates der Go-Toolchain selbst fallen unter die
  reguläre Wartung.
- **Supportzeitraum:** Der Betreiber definiert den unterstützten Zeitraum und
  die unterstützten Go-Versionen (aktuell Go ≥ 1.22).
