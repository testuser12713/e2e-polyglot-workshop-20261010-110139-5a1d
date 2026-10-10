# Kfz-Werkstatt-Kundenportal

Ein Kundenportal für eine Kfz-Werkstatt: Kundinnen und Kunden fragen online einen
Termin an und verfolgen den Status ihres Auftrags, die Werkstatt verwaltet
Kunden, Fahrzeuge und Aufträge und sieht auf einem Dashboard den Arbeitsstand.
Das Produkt besteht aus drei eigenständig laufenden Diensten: einer Go-API
(`net/http`, `pgx`) auf PostgreSQL als einziger Datenbank, einem Python-Worker,
der fertige Aufträge aus einer Valkey-Warteschlange holt und daraus Rechnungen
erzeugt, und einer React/Vite/TypeScript-Web-App, die ausschließlich über die
API mit dem Backend spricht.

## Dienste

| Dienst   | Verzeichnis | Stack                              | Beschreibung                                                              |
| -------- | ----------- | ---------------------------------- | ------------------------------------------------------------------------- |
| `api`    | `backend/`  | Go, `net/http`, `pgx`              | HTTP-API (Basis-Pfad `/api`) auf PostgreSQL                               |
| `worker` | `worker/`   | Python                             | liest die Valkey-Warteschlange und erzeugt Rechnungen (eigener Dienst)    |
| `web`    | `frontend/` | Vite + React + TypeScript          | Web-App mit Kunden- und Werkstattbereich                                  |

## Tech-Stack

- **api**: Go (aktuelle Version), `net/http` aus der Standardbibliothek, `pgx`
  für PostgreSQL.
- **worker**: Python (aktuelle Version), Redis-kompatibler Client für Valkey.
- **web**: Vite + React + TypeScript, Vitest und Testing Library.
- **database**: PostgreSQL 18 — echte Datenbank, kein SQLite.
- **queue**: Valkey 9.1 als Warteschlange.
- **Geldbeträge**: ganze Cent, 19 % Mehrwertsteuer, kaufmännisch gerundet.

## Voraussetzungen

- Go (aktuelle Version)
- PostgreSQL 18 und Valkey 9.1 — entweder aus `compose.yaml` (siehe unten) oder
  von Hand gestartet. Der Dienst verbindet sich beim Start und legt das Schema
  automatisch an.

## Datenbanken starten

Für eine lokale Entwicklungsumgebung liegen in `compose.yaml` die beiden
benötigten Dienste. Das Datenbank-Passwort steht nicht in der Datei, sondern
kommt aus der Umgebung:

```bash
cp .env.example .env      # POSTGRES_PASSWORD und die Geheimnisse eintragen
docker compose up -d
```

Das startet PostgreSQL 18 auf Port `5432` und Valkey 9.1 auf Port `6379`.

## Installation und Start (API)

Die Konfiguration der API kommt vollständig aus Umgebungsvariablen. Eine
Vorlage mit allen Variablen liegt in `.env.example`.

```bash
cd backend
go mod download

export DATABASE_URL="postgresql://app:<POSTGRES_PASSWORD>@localhost:5432/app"
export VALKEY_URL="redis://localhost:6379/0"
export JWT_SECRET="$(openssl rand -hex 32)"
export SEED_EMPLOYEE_EMAIL="chef@workshop.local"
export SEED_EMPLOYEE_PASSWORD="$(openssl rand -hex 16)"
export CORS_ORIGIN="http://localhost:5173"

go run ./cmd/api
```

Die API lauscht danach auf `http://localhost:8000`. Der Health-Endpunkt ist

```bash
curl http://localhost:8000/api/health      # -> 200 {"status":"ok"}
```

Beim Start liest und prüft der Dienst die Konfiguration einmal. Fehlt eine
Pflichtvariable, startet er nicht und nennt die fehlende Variable beim Namen.

## Für Produktion bauen

```bash
cd backend
go build -o bin/api ./cmd/api
./bin/api
```

Die ausgelieferte Binärdatei liest dieselben Umgebungsvariablen.

## Umgebungsvariablen der API

| Variable                 | Pflicht | Bedeutung                                                                 |
| ------------------------ | ------- | ------------------------------------------------------------------------- |
| `DATABASE_URL`           | ja      | PostgreSQL-Verbindung                                                      |
| `VALKEY_URL`             | ja      | Valkey-Verbindung (Warteschlange)                                          |
| `QUEUE_NAME`             | nein    | Name der Valkey-Liste, Standard `workshop:invoices`                        |
| `HTTP_PORT`              | nein    | Port der API, Standard `8000`                                              |
| `JWT_SECRET`             | ja      | Signaturschlüssel der Anmelde-Token (nie im Repository)                    |
| `CORS_ORIGIN`            | ja      | origin der Web-App, dem die API CORS erlaubt                              |
| `SEED_EMPLOYEE_EMAIL`    | ja      | E-Mail des beim Start angelegten ersten Mitarbeiters                       |
| `SEED_EMPLOYEE_PASSWORD` | ja      | Startpasswort dieses Mitarbeiters (nie im Repository)                      |

Diese Werte sind auch in `RUN.json` deklariert. Geheimnisse
(`JWT_SECRET`, `SEED_EMPLOYEE_PASSWORD`) stehen dort nur als Rezept zum
Erzeugen, nie als Wert.

## API benutzen

Basis-Pfad: `/api`. Alle Antworten sind JSON. Werkstatt-Routen verlangen den
Header `Authorization: Bearer <JWT>`.

Jede Fehlerantwort hat denselben Körper:

```json
{ "error": { "code": "not_found", "message": "resource not found" } }
```

Mögliche Codes mit Statuscode: `validation_error` (422), `not_found` (404),
`invalid_transition` (409), `duplicate_plate` (409), `unauthorized` (401),
`rate_limited` (429), `internal_error` (500).

### Endpunkte

| Methode | Pfad                                  | Auth      | Anfragetext                                                                                              | Antwort                          |
| ------- | ------------------------------------- | --------- | -------------------------------------------------------------------------------------------------------- | -------------------------------- |
| GET     | `/api/health`                         | nein      | —                                                                                                        | `{ "status": "ok" }`             |
| POST    | `/api/public/appointments`            | nein      | `{name,email,phone,plate,brand,model,mileageKm,requestedDate,problemDescription}`                        | 201 `{orderNumber,status}`       |
| GET     | `/api/public/orders/{orderNumber}`    | nein      | Query `?plate=<plate>`                                                                                    | 200 `{orderNumber,plate,status,requestedDate,history,invoice}` |
| POST    | `/api/auth/login`                     | nein      | `{email,password}`                                                                                        | 200 `{token,expiresAt}`          |
| POST    | `/api/workshop/customers`             | ja        | `{name,email,phone}`                                                                                      | 201 `Customer`                   |
| GET     | `/api/workshop/customers/{id}`        | ja        | —                                                                                                        | 200 `Customer`                   |
| PUT     | `/api/workshop/customers/{id}`        | ja        | `{name,email,phone}`                                                                                      | 200 `Customer`                   |
| POST    | `/api/workshop/vehicles`              | ja        | `{customerId,plate,brand,model,mileageKm}`                                                                | 201 `Vehicle`                    |
| GET     | `/api/workshop/vehicles/{id}`         | ja        | —                                                                                                        | 200 `Vehicle`                    |
| PUT     | `/api/workshop/vehicles/{id}`         | ja        | `{customerId,plate,brand,model,mileageKm}`                                                                | 200 `Vehicle`                    |
| POST    | `/api/workshop/orders`                | ja        | `{vehicleId,requestedDate,problemDescription}`                                                            | 201 `Order`                      |
| GET     | `/api/workshop/orders/{id}`           | ja        | —                                                                                                        | 200 `Order`                      |
| POST    | `/api/workshop/orders/{id}/items`     | ja        | `{items:[{kind,description,hours?,quantity?,unitPriceCents?}]}`                                           | 200 `Order`                      |
| POST    | `/api/workshop/orders/{id}/status`    | ja        | `{status}`                                                                                                | 200 `Order`                      |
| GET     | `/api/workshop/orders`                | ja        | Query `?status=<status>&plate=<fragment>`                                                                 | 200 `{orders:[OrderSummary]}`    |
| GET     | `/api/workshop/dashboard`             | ja        | —                                                                                                        | 200 `{openCounts,completedToday,monthRevenueGrossCents}` |

Beispieldatenformen:

- `Customer` = `{id,name,email,phone}`
- `Vehicle` = `{id,customerId,plate,brand,model,mileageKm}`
- `Order` = `{id,orderNumber,vehicleId,customerId,plate,brand,model,status,requestedDate,problemDescription,createdAt,items,history}`
- `OrderItem` = `{id,kind,description,hours,quantity,unitPriceCents,amountCents}`
- Auftrags-Status: `requested` → `confirmed` → `in_progress` → `completed` → `picked_up`
  (Oberfläche: angefragt, bestätigt, in Arbeit, fertig, abgeholt). Nur diese
  Reihenfolge ist erlaubt.

### Aktueller Stand dieses Meilensteins

Dieser Stand ist das **Gerüst der Go-API**. Umgesetzt ist der Health-Endpunkt
(`GET /api/health`). Alle übrigen registrierten Routen sind bereits verdrahtet
und antworten sauber mit **501 Not Implemented**, bis ihr jeweiliges Ticket sie
füllt. Die Web-App und der Worker entstehen in ihren eigenen Tickets und eigenen
Verzeichnissen.

## Funktionen

- Kunden mit Name, E-Mail und Telefon anlegen, lesen und ändern.
- Fahrzeuge mit Kennzeichen (normalisiert, eindeutig), Marke, Modell und
  Kilometerstand einem Kunden zuordnen.
- Aufträge mit Fahrzeug, Wunschtermin und Problembeschreibung anlegen und
  Positionen aus Arbeitszeit und Teilen erfassen.
- Statuswechsel nur entlang angefragt → bestätigt → in Arbeit → fertig →
  abgeholt, mit vollständigem Statusverlauf.
- Bei Status „fertig“ landet genau eine Nachricht in der Valkey-Warteschlange;
  der Worker erzeugt daraus eine Rechnung (19 % Mehrwertsteuer, Beträge in
  ganzen Cent) und einen Eintrag im Postausgang.
- Kundenbereich: Terminanfrage und Statusabruf über Auftragsnummer und
  Kennzeichen (ohne personenbezogene Daten des Kunden).
- Werkstattbereich: Anmeldung, Auftragsliste mit Statusfilter und
  Kennzeichen-Suche, Auftragsdetails und Dashboard-Kennzahlen.
- Einheitlicher Fehlerkörper, CORS nur für den konfigurierten Origin und
  Protokolle ohne personenbezogene Daten.
