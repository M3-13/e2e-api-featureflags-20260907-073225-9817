VERDICT: CHANGES_REQUESTED

## Prüfumfang / Einordnung

Reines `go-backend` ohne öffentliche Web-UI. Daraus folgt: Keine Impressums-, Cookie-/Consent-, Widerrufs- oder Barrierefreiheitspflichten für eine Webseite. Geprüft werden daher schwerpunktmäßig DSGVO und EU Cyber Resilience Act (CRA). Der sichtbare Stand enthält kein KI-Modul, daher ist der EU AI Act nicht anwendbar.

---

## 1. DSGVO

### 1.1 Unbegrenzte IP-Bucket-Sammlung im Rate-Limiter
**Schweregrad:** mittel  
**Ort:** `rate_limit.go`, insbesondere `rateLimiter.buckets map[string]*tokenBucket`, `clientIP()` und `rateLimitMiddleware`.

**Befund:** Der Rate-Limiter speichert pro Client-IP dauerhaft einen Token-Bucket. Die Map wächst unbegrenzt; Einträge werden nie entfernt. Aktuell bindet `main.go` den Server an `127.0.0.1:8080`, sodass die dort verarbeiteten `RemoteAddr`-Werte typischerweise Loopback-Adressen und damit regelmäßig keine personenbezogenen Daten sind. Die unbeschränkte Sammlung ist dennoch eine Schwäche: Sie verstößt gegen den Grundsatz der Speicherbegrenzung, sobald der Dienst hinter einem Proxy oder auf einem externen Interface betrieben wird, und schafft einen Ressourcen-DoS-Vektor.

**Behebung:**
- In `rate_limit.go` jedem `tokenBucket` ein `lastSeen time.Time` hinzufügen und in `allow()` aktualisieren.
- In `rateLimiter.allow()` überalterte Buckets entfernen, z. B. nach 15 Minuten Inaktivität, oder die Map auf eine maximale Bucket-Anzahl begrenzen.
- Optional: Bucket-Schlüssel als HMAC/SHA-256 der IP mit geheimem Salt speichern, statt der Roh-IP.
- Ergänzende Tests in `rate_limit_test.go`: überalterte Buckets werden entfernt, Map-Größe bleibt begrenzt.
- Hinweis: Der Rate-Limiter selbst darf für das Produkt erhalten bleiben; die Begrenzung muss nur zeitlich bzw. kapazitiv erfolgen.

### 1.2 Rechtsgrundlage für die flüchtige Nutzerkennung nicht dokumentiert
**Schweregrad:** mittel  
**Ort:** `evaluate_handlers.go` und `middleware.go`.

**Befund:** Die `user`-Kennung wird über den Query-Parameter entgegengenommen, nur im Arbeitsspeicher für die Hash-Berechnung genutzt, nicht im Store gehalten und nicht geloggt. Das ist technisch sauber umgesetzt: `store.go` speichert ausschließlich Flag-Daten, und `loggingMiddleware` protokolliert `r.URL.Path` ohne Query-String. Allerdings fehlt eine dokumentierte Rechtsgrundlage und eine datenschutzrechtliche Einordnung dieser Verarbeitung.

**Behebung:**
- In `README.md` oder `COMPLIANCE.md` aufnehmen:
  - „Die vom Client übergebene Nutzerkennung wird ausschließlich flüchtig im Arbeitsspeicher verarbeitet, um einen deterministischen Feature-Rollout zu berechnen. Sie wird nicht gespeichert, nicht geschrieben und nicht protokolliert. Es dürfen nur pseudonyme Kennungen übergeben werden; echte Namen, E-Mail-Adressen oder Telefonnummern sind unzulässig.“
  - Rechtsgrundlage benennen: berechtigtes Interesse nach **Art. 6 Abs. 1 lit. f DSGVO** an Feature-Steuerung und Stabilität, dokumentiert durch den Betreiber; bei Betrieb als Auftragsverarbeiter zusätzlich Vertrag nach **Art. 28 DSGVO**.
  - Löschkonzept angeben: keine Persistenz, ein Neustart beendet jede Verarbeitung.

### 1.3 Beschreibungstext ohne Längenbegrenzung oder PII-Verbot
**Schweregrad:** niedrig  
**Ort:** `models.go` (`Flag.Description`), `flags_handlers.go` (`flagRequest.Description`, `handleUpdateFlag`).

**Befund:** Das Feld `description` nimmt beliebigen Text bis zur globalen Body-Grenze von 1 MiB an. Es ist kein Verbot dokumentiert, dort personenbezogene Angaben zu speichern, und es fehlt eine fachliche Längenbegrenzung.

**Behebung:**
- In `flags_handlers.go` bei POST und PUT eine sinnvolle Maximallänge setzen, z. B. 1.000 Zeichen, und bei Überschreitung mit `400` und generischer Fehlermeldung antworten.
- In der API-Dokumentation festhalten: „Das Feld `description` darf keine personenbezogenen Daten enthalten.“

### 1.4 Positiv geprüfte Punkte
- Logging enthält keine Query-Strings, keine `user`-Werte, keine IP-Adressen und keine Authorization-Header (`middleware.go`).
- Der In-Memory-Store hält ausschließlich Flag-Daten (`store.go`), erfüllt AC-17.
- Fehlerantworten sind generisch und geben keine internen JSON-Parserfehler oder Stacktraces preis (`flags_handlers.go`, `models.go`), erfüllt AC-14.
- Body-Limit und Content-Type-Pflicht sind umgesetzt (`flags_handlers.go`), erfüllt AC-12/AC-13.

---

## 2. EU Cyber Resilience Act (CRA)

### 2.1 SBOM, Update-Konzept und dokumentierte Sicherheitseigenschaften nicht im sichtbaren Stand
**Schweregrad:** mittel  
**Ort:** `README.md`, `SECURITY.md`, `COMPLIANCE.md` existieren als Dateien, ihr Inhalt ist jedoch nicht Teil des sichtbaren Projektstandes. Im sichtbaren Code sind die technischen Sicherheitsmaßnahmen vorhanden, die zugehörige CRA-Dokumentation ist nicht verifizierbar.

**Befund:** Für ein Produkt mit digitalen Elementen müssen Abhängigkeiten/SBOM, dokumentierte Sicherheitseigenschaften und ein Update-/Patch-Konzept erkennbar sein. Das Fehlen sichtbarer Nachweise ist eine schließbare Lücke.

**Behebung:**
- In `SECURITY.md` oder `COMPLIANCE.md` ergänzen:
  - **SBOM:** Go-Modul, ausschließlich Standardbibliothek (`net/http`, `crypto/*`, `sync`, `time`, `log`, `encoding/json`, `hash/fnv`), Go-Version aus `go.mod`, keine Drittanbieter-Abhängigkeiten.
  - **Sicherheitseigenschaften:** Bearer-Token-Authentifizierung, Body-Limit 1 MiB, Content-Type-Pflicht, generische Fehler, Server-Timeouts, Rate-Limiting, optionale TLS-Nutzung, Bindung an `127.0.0.1`.
  - **Update-/Patch-Konzept:** Versionsschema, Veröffentlichungsweg, Unterstützungszeitraum und Meldeweg für Sicherheitslücken.
  - **Build-/Test-Hinweis:** reproduzierbarer Build mit `go build`, Testausführung mit `go test ./...` und `-race`.

### 2.2 Unbeabsichtigter Klartextbetrieb bei unvollständiger TLS-Konfiguration
**Schweregrad:** niedrig  
**Ort:** `main.go`.

**Befund:** Wenn nur eine der Umgebungsvariablen `TLS_CERT_FILE` oder `TLS_KEY_FILE` gesetzt ist, fällt der Server still auf `ListenAndServe` zurück und startet ohne TLS. Das kann zu einem unbeabsichtigten Klartextbetrieb führen. Die aktuelle Bindung an `127.0.0.1` begrenzt das Risiko erheblich.

**Behebung:**
- In `main.go` prüfen: Wenn genau eine der beiden TLS-Variablen gesetzt ist, mit einer aussagekräftigen Fehlermeldung beenden oder eine Warnung loggen.
- Alternativ eine separate Konfiguration erzwingen, die Klartext nur explizit zulässt.

### 2.3 Positiv geprüfte Punkte
- Server-Timeouts gemäß AC-15 sind vollständig umgesetzt (`main.go`).
- Authentifizierungspflicht für alle Flag- und Evaluate-Endpunkte mit konstantem Zeitvergleich (`middleware.go`).
- Rate-Limiting, Größenbegrenzung und MIME-Prüfung als Sicherheitsmaßnahmen vorhanden.

---

## 3. EU AI Act

Nicht anwendbar. Der sichtbare Stand enthält keine KI-Funktion; es handelt sich um einen deterministischen Feature-Flag-Dienst.

---

## 4. Pflichttexte & UI

Nicht anwendbar. Reines HTTP-Backend ohne öffentliche Benutzeroberfläche. Es entstehen keine Cookie-/Consent-, Impressums- oder Web-Widerrufsbelehrungspflichten. Die notwendige Datenschutzdokumentation für API-Betreiber ist in Abschnitt 1.2 als README-/COMPLIANCE-Ergänzung beschrieben.

---

## 5. Barrierefreiheit

Nicht anwendbar. Kein öffentliches Frontend und keine Web-UI, daher keine WCAG-/BITV-/EAA-Pflichten.

---

## Gesamteinstufung

Keine fundamentalen Rechtsverstöße im Code; die technischen Schutzmaßnahmen sind weitgehend solide. Offen sind behebbare Dokumentations- und Konfigurationslücken: Rechtsgrundlage für die flüchtige Nutzerkennung, Begrenzung der IP-Bucket-Sammlung, Beschränkung des Beschreibungstextes sowie CRA-Artefakte zur SBOM-, Update- und Sicherheitsdokumentation.