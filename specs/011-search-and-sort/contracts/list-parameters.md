# Contract: listing with search and sort

Authoritative: `backend/api/openapi.yaml`, operation `listCollectibles`.

| Parameter | Type | Meaning |
|---|---|---|
| `q` | string, ≤200 | Every whitespace-separated word must appear, case-insensitively, in name, character, series, manufacturer, category, scale, edition or notes. Empty means no search |
| `sort` | `added_desc` (default) · `added_asc` · `name_asc` · `name_desc` · `price_desc` · `price_asc` · `purchased_desc` · `purchased_asc` · `released_desc` · `released_asc` | Missing values last in every order |
| `cursor` | string | Opaque; valid only with the `sort` it was issued for |

400 `validation_failed` on `q` too long, `sort` unknown, or a cursor from another order. Response
unchanged (`CollectionPage`); `totalUnfiltered` still counts the whole vault, so the no-results state
still tells "nothing matches" from "nothing here".
