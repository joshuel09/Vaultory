# Contract: the Trash

Authoritative definitions are in `backend/api/openapi.yaml`; this summarises them.

## `DELETE /api/collectibles/{collectibleId}` (changed meaning, same shape)

204 always, as before (006 FR-025). The collectible now moves to the Trash. If it is already there,
its deletion time is unchanged.

## `GET /api/trash?cursor=&limit=`

200 with a `TrashPage`:

```json
{
  "items": [ { "collectible": { "...a Collectible...": "" }, "deletedAt": "2026-10-25T09:00:00Z", "purgeAt": "2026-11-24T09:00:00Z" } ],
  "nextCursor": "opaque or absent"
}
```

Most recently deleted first. Only the acting collector's items within 30 days. `limit` 1–48,
default 24, matching the gallery.

## `POST /api/trash/{collectibleId}/restore`

200 with the restored `Collectible`. 404 `not_found` for anything not in the acting collector's
Trash within the window: absent, another collector's, already restored, permanently deleted,
expired, or a malformed id.

## `DELETE /api/trash/{collectibleId}`

204 always. Destroys the collectible if it is in the acting collector's Trash; otherwise does nothing.
