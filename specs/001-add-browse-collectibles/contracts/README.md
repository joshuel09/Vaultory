# Contract

The OpenAPI document that lived here now lives at **[`backend/api/openapi.yaml`](../../../backend/api/openapi.yaml)**.

It was written for this feature and stayed here afterwards, which made a repository-wide source of
truth look like a historical artifact of the first feature that happened to need one. By the time it
moved it described the operations of features 001, 006 and 008 and was on version 0.3.0.

Nothing about it changed in the move except its path — the generated types came back byte-identical
(issue #33).

Features 004, 005, 006 and 008 each keep a note in their own `contracts/` folder explaining what
they did or did not add to it.
