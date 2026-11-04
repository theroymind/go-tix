---
paths:
  - "**/*.go"
---

# Go structure

- Group a domain in a package named after it (`internal/contracts/ticket`), short, lowercase and singular.
- No stutter: names read with their package, so `ticket.Status`, not `ticket.TicketStatus`.
- One concept per file where it helps (`status.go`, `priority.go`), with its tests beside it (`status_test.go`). File names are lowercase, with underscores only for suffixes like `_test`.
- Group a domain's settings (limits, defaults, patterns) into one `const (...)` or `var (...)` block in `settings.go`, instead of scattering them across files.
- Test fixtures go in a `testdata/` directory next to the tests that read them.
