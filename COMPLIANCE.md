VERDICT: CHANGES_REQUESTED

# Prüfbericht Feature-Flag-Service (go-backend)

Geprüft wird der gemergte Produktstand. Der Service ist eine reine REST-API ohne Endnutzer-UI. Pflichttexte, Cookie-/Consent-Banner und Barrierefreiheit sind daher **nicht anwendbar**. Relevant sind **DSGVO** und – sofern der Service als Produkt mit digitalen Elementen in Verkehr gebracht wird – der **EU Cyber Resilience Act (CRA)**. Der **EU AI Act** ist mangels KI-Funktion nicht einschlägig.

Die Acceptance Criteria sind im sichtbaren Code weitgehend erfüllt. Die Datenschutz-Basismaßnahmen sind solide: kein Logging des `user`-Query-Strings, keine Speicherung von Nutzerkennungen, saubere Fehlerbehandlung, Body-Limit und Server-Timeouts. Für die Marktreife fehlen jedoch Authentifizierung, Transportverschlüsselung und die nach CRA erforderliche Sicherheitsdokumentation.

---

## 1. DSGVO / GDPR

### Befund 1.1 — Rechtsgrundlage für die Verarbeitung der Nutzerkennung nicht dokumentiert
**Schweregrad:** medium

Der Query-Parameter `user` in `GET /flags/{key}/evaluate` ist ein personenbezogenes Datum. Der Code verarbeitet ihn ausschließlich transient: Er wird gehasht, nicht gespeichert und nicht geloggt. Das ist datenschutzfreundlich, aber die Rechtsgrundlage (z. B. Art. 6 Abs. 1 lit. b oder lit. f DSGVO) und die Verantwortlichkeit sind im Repo nicht dokumentiert.

**Konkrete Abhilfe:**
- `README.md` (vorhanden) um einen Abschnitt „Datenschutz & Rechtsgrundlage“ ergänzen:
  - Verarbeitete Daten: `user`-ID bei der Evaluierung.
  - Rechtsgrundlage benennen.
  - Hinweis, dass keine persistente Speicherung erfolgt.
  - Betreiberpflicht: Aufnahme in das Verarbeitungsverzeichnis, ggf. DSFA-Prüfung.

### Befund 1.2 — Transport ohne TLS, Nutzerkennung im Klartext übertragbar
**Schweregrad:** high

`main.go` startet den Server auf `:8080` ausschließlich über `http.ListenAndServe`. Es gibt keinen TLS-Listener und keinen sichtbaren Zwang, den Dienst hinter einem TLS-terminierenden Proxy zu betreiben. Die `user`-ID wird dadurch im Query-String und in der JSON-Antwort potenziell unverschlüsselt über das Netz übertragen. Das verletzt die Anforderungen aus Art. 32 DSGVO an geeignete technische und organisatorische Maßnahmen zur Sicherstellung von Vertraulichkeit und Integrität, wenn der Dienst über nicht vertrauenswürdige Netze erreichbar ist.

**Konkrete Abhilfe:**
- `main.go`: optional TLS-Unterstützung ergänzen, z. B. über Umgebungsvariablen `TLS_CERT_FILE`/`TLS_KEY_FILE` und `tls.ListenAndServeTLS`.
- `README.md`: verbindliche Betriebsvorgabe aufnehmen: „Der Dienst darf nur hinter TLS-Terminierung (Reverse Proxy) oder mit direktem TLS betrieben werden; kein öffentlicher Betrieb über unverschlüsseltes HTTP.“

### Befund 1.3 — Access-Logs ohne definierte Retention/Rotation
**Schweregrad:** low

`middleware.go` schreibt Logs nach `stdout`. Die Logs enthalten dank `r.URL.Path` ohne Query-String keine `user`-Werte. Eine Aufbewahrungs- oder Rotationsregel ist jedoch nicht dokumentiert.

**Konkrete Abhilfe:**
- `README.md`: Abschnitt „Betrieb / Logging“ ergänzen: Logs nach `stdout`, keine PII, Aufbewahrung/Rotation durch die Betriebsumgebung definieren.

### Befund 1.4 — `user` wird unnötig in der Antwort zurückgespiegelt
**Schweregrad:** low

`evaluate_handlers.go` liefert in `evaluateResult` das Feld `User` an den Client zurück. Der Aufrufer kennt die übergebene `user`-ID bereits. Die Rückspiegelung ist datenschutzrechtlich nicht verboten, aber nicht erforderlich und widerspricht dem Grundsatz der Datenminimierung.

**Konkrete Abhilfe:**
- `evaluate_handlers.go`: Feld `User string \`json:"user"\`` aus `evaluateResult` entfernen, sofern keine API-Kompatibilität zwingend dagegenspricht.
- Dazugehörige Tests in `evaluate_handlers_test.go` entsprechend anpassen (`res1.User`-Asserts entfernen).

---

## 2. EU Cyber Resilience Act (CRA)

### Befund 2.1 — Keine Authentifizierung/Authorization für mutierende Endpunkte
**Schweregrad:** high

`POST /flags`, `PUT /flags/{key}` und `DELETE /flags/{key}` sind ohne Authentifizierung erreichbar. Jeder Netzzugriff kann Flags anlegen, verändern oder löschen. Auch `GET /flags` und `GET /flags/{key}/evaluate` sind ungeschützt, wodurch Geschäftsdaten und Evaluierungsergebnisse offenliegen. Das widerspricht „security by design/default“ im Sinne des CRA und dem DSGVO-Grundsatz der Integrität und Vertraulichkeit.

**Konkrete Abhilfe:**
- Neue `authMiddleware` in `middleware.go` implementieren: Prüfung eines konfigurierbaren API-Keys/Bearer-Tokens.
- In `main.go`: Authentifizierung auf alle Routen außer `GET /healthz` anwenden.
- Konfiguration über Umgebungsvariable, z. B. `AUTH_TOKEN`, mit sicherem Default: Fehler `401` und generisches JSON-Fehlerobjekt.
- `middleware_test.go` und Handler-Tests um Auth-Fälle ergänzen.
- **Vereinbarkeit:** Legitime Clients erhalten den Token; `healthz` bleibt öffentlich. Damit funktioniert das Produkt unter der eigenen Sicherheitsanforderung weiterhin.

### Befund 2.2 — Kein SBOM / keine dokumentierte Abhängigkeitsliste
**Schweregrad:** medium

Der CRA verlangt für Produkte mit digitalen Elementen eine nachvollziehbare Auflistung der enthaltenen Komponenten (SBOM). Das Projekt nutzt laut sichtbarem Code ausschließlich die Go-Standardbibliothek; dokumentiert ist das aber nicht.

**Konkrete Abhilfe:**
- `README.md` um einen Abschnitt „SBOM / Abhängigkeiten“ ergänzen: Sprache Go, Standardbibliothek, keine externen Module; `go.mod` enthält nur Modulname und Go-Version.
- Optional eine maschinenlesbare SBOM-Datei (`sbom.spdx.json` oder CycloneDX) im Repo ergänzen.

### Befund 2.3 — Sicherheitseigenschaften nicht dokumentiert
**Schweregrad:** medium

Der Code implementiert wichtige Sicherheitsmaßnahmen: Timeouts, 1-MiB-Body-Limit, Content-Type-Prüfung, generische Fehlertexte, Logging ohne Query-String, thread-sicheren Store. Eine Dokumentation dieser Eigenschaften fehlt.

**Konkrete Abhilfe:**
- `README.md` um einen Abschnitt „Security Properties“ ergänzen:
  - Server-Timeouts (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`) mit Werten.
  - Body-Limit (`1 MiB`) und `Content-Type: application/json`-Pflicht.
  - Fehlerbehandlung: keine internen Fehlertexte an Clients.
  - Logging: Pfad ohne Query-String, keine Nutzerkennungen.
  - Datenhaltung: ausschließlich Flag-Daten im flüchtigen In-Memory-Store.

### Befund 2.4 — Update-/Patchfähigkeit nicht beschrieben
**Schweregrad:** low

Für den CRA ist ein dokumentierter Prozess für Sicherheitsupdates erforderlich.

**Konkrete Abhilfe:**
- `README.md`: Abschnitt „Wartung und Updates“ ergänzen: Verantwortlichkeit, Versionshistorie, Verfahren für Sicherheits-Patches, Supportzeitraum.

---

## 3. EU AI Act

**Nicht anwendbar.** Der Service enthält keine KI-Funktion im Sinne der Verordnung.

---

## 4. Pflichttexte & UI

**Nicht anwendbar.** Es handelt sich um eine reine REST-API ohne Endnutzer-UI. Impressum, AGB, Datenschutzerklärung für Endnutzer und Cookie-Banner sind auf Ebene dieses Backends nicht erforderlich; sie müssen im zugehörigen Frontend oder im Vertragsverhältnis des Betreibers erfüllt werden.

---

## 5. Barrierefreiheit

**Nicht anwendbar.** Keine öffentliche Weboberfläche, daher keine WCAG/BITV/EAA-Pflicht.

---

## Fazit

Der Code erfüllt die spezifizierten Funktions- und Sicherheitskriterien inklusive der datenschutzfreundlichen Logging- und Speicherpraxis. Offene rechtliche Blocker im Sinne einer sofortigen Untersagung bestehen nicht. Vor einer Marktfreigabe sind jedoch mindestens Authentifizierung für die API, Transportverschlüsselung sowie die CRA-/DSGVO-Dokumentation nachzurüsten. Daher: **CHANGES_REQUESTED**.