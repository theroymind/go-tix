---
type: idea
status: open
summary: Move the strict single-document YAML decode, the two-space encode and the atomic write into one package both tix and sketch import, now that each app holds its own copy.
---

# Share the YAML and atomic-write helpers across apps

The review of [[sketch-schema-and-store]] found that `internal/sketch/store` repeats what `internal/tix/backend` already does: decode one YAML document with known fields only, encode with a two-space indent, and write through a temporary file and a rename. The rule that no app imports another app forced the copy. With two apps holding it, a shared package outside any app, such as `internal/yamlfile` or a home in `railyard-cli`, would keep the two from drifting. The sketch copy also gives each write a unique temporary name through `os.CreateTemp`, which the tix one could adopt.
