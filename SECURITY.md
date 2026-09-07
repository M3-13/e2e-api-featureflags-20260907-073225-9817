VERDICT: CHANGES_REQUESTED

## Scanner-Abdeckung
Es wurden keine anwendbaren Security-Scanner-Ergebnisse geliefert (`no applicable security scanners for this project type`). Das ist eine Dokumentationslücke, aber kein eigenständiger Befund. Die folgende Bewertung basiert auf manueller Code-Analyse des sichtbaren Go-Backends.

## Zusammenfassung
Die Anwendung erfüllt die wesentlichen Security-Vorgaben: keine hartkodierten Secrets, Body-Limit von 1 MiB, Content-Type-Prüfung, generische JSON-Fehler, Timeouts, Auth-Middleware mit konstantem Token-Vergleich und ein Logging, das Query-Strings ausblendet. Es besteht jedoch eine relevante Verfügbarkeitsschwäche in der Reihenfolge von Rate-Limiting und Authentifizierung sowie zwei kleinere Härtungsthemen.

---

## Befund 1 — Rate-Limit vor Authentifizierung ermöglicht unauthentifizierte Denial-of-Service-Angriffe

**Schweregrad:** medium  
**Betroffene Stelle:** `main.go` – `Handler: loggingMiddleware(rateLimitMiddleware(mux))` in Kombination mit `newMux()` (Auth nur innerhalb der einzelnen Routen).

**Beschreibung:**  
Das Rate-Limiting wird um den gesamten Mux gelegt und zählt daher auch unauthentifizierte Anfragen, die von `authMiddleware` später mit `401` beantwortet werden. Dadurch kann ein Angreifer den Token-Bucket leeren, ohne gültige Anmeldedaten zu besitzen.

Da der Server explizit auf `127.0.0.1:8080` lauscht, ist ein Betrieb hinter einem Reverse-Proxy (z. B. auf demselben Host) sehr wahrscheinlich. In dieser üblichen Konstellation ist `RemoteAddr` für alle eingehenden Verbindungen die Proxy-IP (oft `127.0.0.1`). Damit teilen sich **alle Clients** einen einzigen Token-Bucket. Bereits 100 schnelle, unauthentifizierte Anfragen genügen, um den Bucket zu leeren; anschließend erhalten auch legitime authentifizierte Anfragen `429 Too Many Requests`.

**Konkreter Fix:**  
Die Reihenfolge von Auth und Rate-Limit umkehren, z. B. für geschützte Routen:

```go
mux.Handle("GET /flags", authMiddleware(rateLimitMiddleware(http.HandlerFunc(handleListFlags))))
```

`/healthz` bleibt bewusst ungeschützt. Für `/healthz` kann ein eigener, kleiner Rate-Limit-Limiter verwendet oder der Endpoint in der lokalen Standardkonfiguration ohne Limit betrieben werden. Wichtig ist, dass unauthentifizierte `401`-Antworten den Bucket für authentifizierte Nutzer nicht verbrauchen.

**Reconciliation:**  
Die Änderung erhält die Rate-Limit-Funktion für authentifizierte Clients vollständig. `/healthz` bleibt erreichbar; authentifizierte Feature-Flag-Endpunkte funktionieren unverändert, sind aber nicht mehr durch unauthentifizierte Floods aus derselben Proxy-IP blockierbar.

---

## Befund 2 — Token-Bucket-Map wächst unbegrenzt

**Schweregrad:** low  
**Betroffene Stelle:** `rate_limit.go` – `rateLimiter` mit `buckets map[string]*tokenBucket`.

**Beschreibung:**  
Für jede neue `clientIP` wird ein Bucket in der Map angelegt, aber niemals entfernt. In der aktuellen lokalen Konfiguration ist die Angriffsfläche klein, da nur wenige IPs vorkommen. Sobald der Dienst jedoch hinter dem Proxy oder in anderer Umgebung mit vielen Client-IPs betrieben wird, wächst die Map ungebremst und kann Speicher erschöpfen.

**Konkreter Fix:**  
Einen periodischen Cleanup einführen, der Buckets mit sehr altem `lastRefill` löscht, oder die Map-Größe begrenzen. Beispiel: ein Hintergrund-`time.Ticker` im `rateLimiter`, der Einträge ohne Aktivität seit z. B. 10 Minuten entfernt. Alternativ ein LRU-Ansatz für die Buckets.

**Reconciliation:**  
Das normale Rate-Limit-Verhalten bleibt unverändert; lediglich verwaiste Buckets werden entfernt. Die Funktionalität der Endpunkte wird nicht beeinträchtigt.

---

## Befund 3 — Log-Injection über URL-Pfad möglich

**Schweregrad:** low  
**Betroffene Stelle:** `middleware.go` – `logger.Printf("method=%s path=%s status=%d duration=%s", r.Method, r.URL.Path, rec.status, time.Since(start))`.

**Beschreibung:**  
`r.URL.Path` ist URL-dekodiert und kann Steuerzeichen wie Zeilenumbrüche enthalten (z. B. `%0A`). Ein Angreifer kann dadurch manipulierte Anfragen senden, die zusätzliche Log-Zeilen vortäuschen oder Log-Analyse-Systeme stören. Der Query-String wird bereits korrekt entfernt; der Pfad selbst wird jedoch ungefiltert protokolliert.

**Konkreter Fix:**  
Den Pfad beim Logging sanitisieren, z. B. durch eine Hilfsfunktion:

```go
func logSafe(s string) string {
    return strings.Map(func(r rune) rune {
        if r < 0x20 || r == 0x7f {
            return '_'
        }
        return r
    }, s)
}
```

und dann:

```go
logger.Printf("method=%s path=%s status=%d duration=%s",
    r.Method, logSafe(r.URL.Path), rec.status, time.Since(start))
```

**Reconciliation:**  
Normale Pfade wie `/flags` oder `/flags/key/evaluate` bleiben exakt gleich und bestehen keine Log-Tests auf `path=/flags` unverändert. Nur nicht druckbare Zeichen werden ersetzt; die Produktfunktion wird nicht beeinflusst.

---

## Nicht beanstandet (geprüft, aber unauffällig)
- **Secrets:** `AUTH_TOKEN` wird ausschließlich aus der Umgebungsvariable gelesen; keine hartkodierten Produktions-Secrets. Der Test-Token `test-token` ist als Test-Fixture klar erkennbar und unkritisch.
- **Injection:** JSON-Parsing-Fehler werden generisch behandelt; Body-Limit und Content-Type-Prüfung sind vorhanden. Keine SQL-, Command- oder Pfad-Injection erkennbar.
- **AuthN/AuthZ:** Der Bearer-Token-Vergleich erfolgt mit `subtle.ConstantTimeCompare`; ein leerer `AUTH_TOKEN` führt zu fail-closed `401`.
- **Dependencies:** Es sind nur Standardbibliotheks-Pakete im Einsatz; keine bekannten verwundbaren Drittanbieter-Abhängigkeiten sichtbar.
- **Transport/Timeouts:** Der Server ist auf `127.0.0.1:8080` gebunden und mit `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` und `IdleTimeout` konfiguriert. TLS ist optional, was in der lokalen Standardkonfiguration vertretbar ist.