---
type: idea
status: open
summary: A lint rule that flags list items inside a note's Description section, which the Writing tickets rules keep as prose.
---

# Lint a Description that holds a list

The Writing tickets rules in SPEC.md say a description is prose, with no lists, because the criteria hold the checklist. `tix lint` could enforce it with a `prose` rule over the sections a profile names, such as `prose_sections: [Description]`, flagging any list item inside them.
