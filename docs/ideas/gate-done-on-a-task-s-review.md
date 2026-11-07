---
type: idea
status: open
summary: Have tix done refuse a task whose review property is still pending, and tix lint flag a review value other than none, pending or passed.
---

# Gate done on a task's review

A task's `review` property says whether it needs a review and whether it has had one: `none` when it needs none, `pending` from planning until the review's findings are settled, and `passed` once they are. It replaced the `true` and `false` values while the first sketch schema and store were built, keeping review apart from the status, which tix v1 maps to its five statuses. Today nothing reads it, so `tix done` could refuse a `pending` task the way it refuses one with an unticked criterion, with `--force` to override, and `tix lint` could check the value is one of the three. The property and its values would belong in the profile, so a vault without reviews names none.
