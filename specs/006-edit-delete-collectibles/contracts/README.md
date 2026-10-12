# Contracts: Edit & Delete Collectibles

The contract for this feature is **not** in this directory. It is in
[`specs/001-add-browse-collectibles/contracts/openapi.yaml`](../../001-add-browse-collectibles/contracts/openapi.yaml),
now at version **0.2.0**.

That document is the single source of truth for the frontend/backend seam (Constitution Principle
III) and is what `npm run generate:api` reads. Splitting it per feature would give one seam two
sources, and the generator would then read only one of them. Features 004 and 005 set the same
precedent: a `contracts/` README pointing at where the real contract lives.

Moving the document somewhere neutral — `backend/api/openapi.yaml`, say — is the right end state,
but it is a repository-wide change unrelated to editing and deleting, and Principle V keeps it out
of this feature's scope.

## What 0.2.0 added

One new path, `/collectibles/{collectibleId}`, with three operations:

| Operation | Method | Notes |
|---|---|---|
| `getCollectible` | `GET` | Fills the edit screen. Same representation as a gallery entry — one shape, so list and detail cannot drift. 404 for another collector's (FR-031). |
| `editCollectible` | `PUT` | Full replacement. 409 on a stale `expectedVersion`, carrying the current collectible (FR-027). |
| `deleteCollectible` | `DELETE` | Always `204` for an authenticated collector, whether or not anything was there (FR-025). No 404 is defined, deliberately. |

Two new schemas — `EditCollectibleRequest` and `VersionConflictResponse` — and one new reusable
response, `VersionConflict`.

`Collectible` gained a required `version` property. It appears on every collectible the API returns,
including in the gallery, because there is one projection rather than a separate detail shape.

## The two things most likely to be got wrong

**`imageId` on an edit is a full replacement, not a patch.** Omitted or null means *this collectible
has no photograph*, and removes the one it had (FR-017). A client that is not touching the
photograph must send the current image's id back (FR-018). An integration test asserts that an edit
changing only the name keeps the image, because this is the failure that would otherwise be noticed
by a collector rather than by the suite.

**`DELETE` has no 404.** An identifier that is not in the acting collector's vault is answered `204`
like any other. That is what makes a retry safe and what makes another collector's identifier
indistinguishable from a fictional one. Adding a 404 later would break both.
