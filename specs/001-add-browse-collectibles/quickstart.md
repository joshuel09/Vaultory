# Quickstart & Validation: Add & Browse Collectibles

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Date**: 2026-09-07

How to run this feature locally and prove it satisfies the specification. Every scenario below maps
to numbered acceptance scenarios or requirements in `spec.md`, so a reviewer can walk the list and
see the feature working. Schemas and status codes are defined in
[`contracts/openapi.yaml`](./contracts/openapi.yaml); field rules and constraints are in
[`data-model.md`](./data-model.md). Neither is repeated here.

---

## Prerequisites

- Go — the version pinned in `backend/go.mod`
- Node.js LTS and a package manager, per `frontend/package.json`
- PostgreSQL 16 or newer, reachable locally
- `golang-migrate`, for applying schema migrations
- `curl` and `jq`, for the request walkthroughs below

## Environment

The backend reads all configuration from the environment; nothing secret is committed
(Principle IV). At minimum:

| Variable | Purpose |
|----------|---------|
| `VAULTORY_DATABASE_URL` | PostgreSQL connection string |
| `VAULTORY_IMAGE_STORE` | `filesystem` for local development |
| `VAULTORY_IMAGE_STORE_PATH` | Directory for stored originals and renditions |
| `VAULTORY_DEV_IDENTITY` | `enabled` — activates the development-only collector resolution (research Decision 1). Never enabled outside development |
| `VAULTORY_LISTEN_ADDR` | Address the Go service listens on |

The frontend needs the backend's address for its `/api/*` rewrite. Confirm the rewrite's body limit
is above 10 MB, or uploads fail before the backend can apply the size rule (research Decision 2).

## Setup

```bash
# 1. Database and schema
createdb vaultory_dev
migrate -path backend/migrations -database "$VAULTORY_DATABASE_URL" up

# 2. Backend
cd backend && go run ./cmd/vaultory-api

# 3. Frontend, in a second shell
cd frontend && npm install && npm run dev
```

Migrations seed one development collector. Open the app and use the development sign-in path to
become that collector; the session cookie it sets is what every request below relies on. Save it for
`curl`:

```bash
# Adjust to the development sign-in path exposed by the backend
curl -s -c /tmp/vaultory.jar -X POST http://localhost:3000/api/dev/session
```

All `curl` examples go through the frontend's origin (port 3000), not the backend directly — that is
how the browser talks to it, and it is what makes the session cookie present on image requests.

---

## Walkthrough A — Add a collectible with only what is required

Covers User Story 1 scenario 1, FR-002, FR-005.

```bash
curl -s -b /tmp/vaultory.jar -X POST http://localhost:3000/api/collectibles \
  -H 'Content-Type: application/json' \
  -d '{"name":"Kaiju Sentinel","collectionStatus":"owned"}' | jq
```

Expect `201` and a `Collectible` whose `id` is set, `image` is null, and every optional attribute is
null. **This is the core proof**: a name and a status alone are sufficient.

## Walkthrough B — Add a collectible with every attribute

Covers User Story 1 scenario 2, FR-006, FR-007, FR-016.

Upload an image first, then reference it:

```bash
IMAGE_ID=$(curl -s -b /tmp/vaultory.jar -X POST http://localhost:3000/api/images \
  -F file=@./sentinel.jpg | jq -r .id)

curl -s -b /tmp/vaultory.jar -X POST http://localhost:3000/api/collectibles \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Kaiju Sentinel\",\"collectionStatus\":\"owned\",\"character\":\"Sentinel Prime\",
       \"series\":\"Kaiju Wars\",\"manufacturer\":\"Apex Studio\",\"category\":\"Statue\",
       \"scale\":\"1/4\",\"edition\":\"Deluxe Exclusive\",\"purchasePrice\":\"1250.00\",
       \"purchaseDate\":\"2026-08-14\",\"releaseDate\":\"2026-11-30\",
       \"notes\":\"Box has a small dent on the lower left corner.\",\"imageId\":\"$IMAGE_ID\"}" | jq
```

Expect `201` with every value echoed back, `purchasePrice` exactly `"1250.00"` as a string, and
`image.renditionUrl` populated. Verify the price is byte-identical to what was sent — no float
conversion anywhere.

## Walkthrough C — Validation reports every problem at once

Covers User Story 1 scenarios 3 and 4, FR-002, FR-003, FR-017, FR-018, FR-020.

```bash
curl -s -b /tmp/vaultory.jar -X POST http://localhost:3000/api/collectibles \
  -H 'Content-Type: application/json' \
  -d '{"name":"   ","collectionStatus":"borrowed","purchasePrice":"-5.00","purchaseDate":"2030-01-01"}' \
  -w '\n%{http_code}\n' | jq
```

Expect `400` and a single `ErrorResponse` whose `error.fields` contains **four** entries at once —
`name` required (whitespace only is a missing name), `collectionStatus` invalid, `purchasePrice`
negative, `purchaseDate` in the future. One problem per request would fail FR-020.

## Walkthrough D — A rejected image does not block saving

Covers User Story 1 scenario 6, FR-009, FR-010, FR-013.

```bash
# Oversized: generate a file over 10 MB
head -c 12000000 /dev/urandom > /tmp/too-big.jpg
curl -s -b /tmp/vaultory.jar -X POST http://localhost:3000/api/images \
  -F file=@/tmp/too-big.jpg -w '\n%{http_code}\n' | jq

# Wrong format: a PDF renamed as an image
curl -s -b /tmp/vaultory.jar -X POST http://localhost:3000/api/images \
  -F file=@./document.pdf\;type=image/jpeg -w '\n%{http_code}\n' | jq
```

Expect `413` with `image_too_large` and a message stating the 10 MB limit, then `400` with
`unsupported_image_format` and a message naming JPEG, PNG, and WebP. The declared
`type=image/jpeg` must not help — the decode decides (research Decision 4).

Then confirm the collectible still saves with no image at all: repeat Walkthrough A. Nothing about a
refused upload may block it.

## Walkthrough E — Browse the collection

Covers User Story 2 scenarios 1, 2, 6; FR-030, FR-033, FR-034, FR-035.

```bash
curl -s -b /tmp/vaultory.jar 'http://localhost:3000/api/collectibles?limit=24' | jq
```

Expect `200` with `items` newest-first, `totalUnfiltered` set, and `nextCursor` present only when
more remain. Page again with `?cursor=<nextCursor>` and confirm no entry repeats or is skipped.

In the browser at `/collection`, confirm by eye:

- imagery dominates each card; the view is a gallery, not a table (FR-030, FR-031, SC-007)
- a collectible with no image shows the designed placeholder, not a broken image (FR-033)
- every card frames its image identically, whatever the original proportions (FR-014, SC-013)
- name and status are both readable on each card (FR-032), and status is not signalled by colour
  alone (FR-044)
- the layout holds at desktop, tablet, and mobile widths with no horizontal page scroll (SC-010)

## Walkthrough F — Filter by status

Covers User Story 3, FR-037 to FR-040.

```bash
curl -s -b /tmp/vaultory.jar 'http://localhost:3000/api/collectibles?status=preordered' | jq '.items[].collectionStatus'
curl -s -b /tmp/vaultory.jar 'http://localhost:3000/api/collectibles?status=sold' | jq '{items: (.items|length), totalUnfiltered}'
```

Expect only `preordered` in the first, and in the second — with no sold entries — `items` empty
while `totalUnfiltered` stays above zero. That difference is what lets the frontend distinguish a
filter matching nothing from an empty vault. In the browser, confirm the no-results state differs
visibly from the empty state, and that the active filter is indicated (FR-039, FR-040).

## Walkthrough G — Privacy holds

Covers FR-015, FR-026, FR-027, FR-029; SC-005, SC-012. **The most important walkthrough here.**

```bash
# 1. No session at all
curl -s http://localhost:3000/api/collectibles -w '\n%{http_code}\n'

# 2. A second collector's session must not see the first collector's image
curl -s -c /tmp/other.jar -X POST http://localhost:3000/api/dev/session?collector=second
curl -s -b /tmp/other.jar "http://localhost:3000/api/images/$IMAGE_ID/rendition" -w '\n%{http_code}\n'

# 3. The owner can
curl -s -b /tmp/vaultory.jar "http://localhost:3000/api/images/$IMAGE_ID/rendition" \
  -o /dev/null -w '%{http_code} %{content_type}\n'
```

Expect `401` with no collection content; then `404` — **not** `403`, which would confirm the image
exists (FR-027); then `200` with `image/jpeg`. Also confirm the second collector's listing never
contains the first collector's entries.

## Walkthrough H — Duplicates stay independent

Covers User Story 1 scenario 5, FR-023, FR-024.

Run Walkthrough A twice, then list. Expect two entries with distinct `id`s, both present. Add a
third with the same name but `"collectionStatus":"sold"` and a different `purchasePrice`; confirm all
three coexist with their own values and that nothing is merged or shown as a quantity.

---

## Automated test suites

```bash
cd backend  && go test ./...          # unit, integration, contract
cd frontend && npm run test           # Vitest + Testing Library
cd frontend && npm run test:e2e       # Playwright
```

What each layer must cover, mapped to the specification:

| Layer | Coverage |
|-------|----------|
| Backend unit (`internal/domain`, `internal/imaging`) | Every validation rule in `data-model.md`: required name including whitespace-only, the four statuses, exact-decimal money including zero and negative and excess precision, date bounds, length caps, all-problems-at-once. Image decode, format refusal, EXIF orientation, fixed-aspect rendition. No database or HTTP involved (Principle II) |
| Backend integration (real PostgreSQL) | Ownership isolation as a first-class test: a second collector can reach neither collectible nor image, and receives `404` rather than `403`. Keyset pagination with no repeats or gaps under concurrent inserts. Status filtering. `NUMERIC` round-trip exactness. Duplicate entries staying independent. Migrations apply and reverse cleanly |
| Backend contract | Every response conforms to `contracts/openapi.yaml`, including the error envelope and each documented status code |
| Frontend unit | Each interface state renders and is distinguishable: empty, loading, error with retry, no-results, success, validation. Card shows name and status; placeholder for missing image; status conveyed without relying on colour |
| Frontend e2e | The whole journey — add a collectible, see it in the gallery, filter to its status, filter to a status with no matches — plus the 10 MB refusal followed by a successful save with no image |

## Quality gates

The constitution requires all of these to pass before this feature is complete:

```bash
cd backend  && go vet ./... && go build ./... && go test ./...
cd frontend && npm run lint && npx tsc --noEmit && npm run build && npm run test
```

Additionally: the OpenAPI contract reflects every operation, the frontend's generated types are
regenerated from it, schema changes ship as reversible migrations, and the acceptance scenarios above
are demonstrably satisfied.

## Troubleshooting

| Symptom | Likely cause |
|---------|--------------|
| Images 404 in the browser but `curl` with a cookie works | The `<img>` is bypassing the same-origin rewrite, so no session cookie is sent. Check `next.config.ts` and that `next/image` optimization is not fetching server-side (research Decision 2) |
| A 10 MB upload fails before the backend responds | The rewrite proxy's body limit is below 10 MB |
| Photographs of figures appear rotated | EXIF orientation is not being applied before the rendition is derived |
| `purchasePrice` comes back as a number, or a cent is lost | Money is being handled as a float somewhere; it must stay a string across the contract and `NUMERIC` in the database |
| Another collector's request returns `403` | Existence is being revealed; FR-027 requires `404` |
