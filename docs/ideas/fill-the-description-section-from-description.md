---
type: idea
status: open
summary: On the vault backend, tix add --description writes under the H1 and leaves a template's Description placeholder in place.
---

# Fill the Description section from --description

Found while migrating go-tix's own tracker to the starter vault. The starter task and feature templates have a `## Description` section, but `tix add --description` puts the text in a paragraph under the H1, so the placeholder stays and the note carries two descriptions. When the template has a section under the profile's description heading, the text should replace that section's body. That needs a `description_heading` in the profile, or treating `Description` as the default, and `tix show` reading the same section back.
