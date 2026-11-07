# tix

A per-repo ticket tracker for humans and coding agents, and `sketch`, a file explorer for planning a feature's files in notes beside them. Tickets are notes in an Obsidian vault, the default, or YAML files in `.tix/`, committed with the code. `tix ready` answers "what do I pick up next?". See [docs/tix/SPEC.md](docs/tix/SPEC.md).

## Install

```bash
go install github.com/theroymind/go-tix/cmd/tix@latest
```

## Use

```bash
tix init --prefix ab            # a starter vault in docs/; --backend yaml for v1's .tix/*.yml
id=$(tix add --type task --acceptance "..." "title")
tix ready
tix show "$id"
tix done "$id"
```

## Sketch

`sketch` is a terminal file explorer for roughing out a feature before it's built. To plan a file, drop a note beside it named after it with `.sketch.md` added: `src/composables/useOrbit.ts.sketch.md` plans `src/composables/useOrbit.ts`. A note for a file that doesn't exist yet shows in the tree as a `+` ghost node, and an existing file with a note shows as `~`.

```bash
go install github.com/theroymind/go-tix/cmd/sketch@latest
sketch                    # the explorer; a add · A attach · e edit · d delete · r ref · f feature · ? question · n notes · R reload
sketch ls --json          # every note with its target, marks, items and questions, for agents
sketch questions --json   # the open questions
```

A note is free text. These lines give it structure, and `sketch help` has the details:

```markdown
---
feature: camera-controls
---
# useOrbit.ts
#fn useOrbit target Ref<Vec3>
? clamp pitch here or in useCamera
> in useCamera, it owns the limits
```

`feature:` and `task:` link the note to tix tickets. `#fn`, `#type`, `#component`, `#hook`, `#flow`, `#tune`, `#rule` and `#test` name an item. A `?` line is an open question until a `>` line right under it answers it.

In LazyVim, this spec in `~/.config/nvim/lua/plugins/sketch.lua` opens it in a float on `<leader>k`, as LazyVim opens lazygit. `e` in the float hides it and opens the note in that nvim, and the next `<leader>k` brings the explorer back. The explorer reloads when its terminal regains focus, and `R` reloads it by hand.

```lua
local term

local function toggle()
  term = Snacks.terminal.toggle(vim.g.sketch_cmd or "sketch", {
    cwd = LazyVim.root(),
    win = { position = "float", width = 0.9, height = 0.9, border = "rounded" },
  })
end

-- sketch's e runs nvim --server $NVIM --remote: hide the float and focus the note it opened.
local function move_note_out_of_float(event)
  if not (term and term.win and vim.api.nvim_win_is_valid(term.win)) or vim.bo[event.buf].buftype ~= "" then
    return
  end
  vim.schedule(function()
    if vim.api.nvim_win_get_buf(term.win) ~= term.buf then
      vim.api.nvim_win_set_buf(term.win, term.buf)
    end
    term:hide()
    local note_win = vim.fn.bufwinid(event.buf)
    if note_win ~= -1 then
      vim.api.nvim_set_current_win(note_win)
    else
      vim.api.nvim_set_current_buf(event.buf)
    end
  end)
end

return {
  "folke/snacks.nvim",
  keys = { { "<leader>k", toggle, desc = "Sketch explorer" } },
  init = function()
    vim.api.nvim_create_autocmd("BufWinEnter", {
      group = vim.api.nvim_create_augroup("sketch_float", { clear = true }),
      callback = move_note_out_of_float,
    })
  end,
}
```

The `#TYPE` lines read as headings without a space to markdownlint, and a note needn't start with a heading. With markdownlint-cli2 0.23.0 or later, as LazyVim's markdown extra runs, this `.markdownlint-cli2.yaml` at the repo root turns off those two rules for notes only:

```yaml
overrides:
  - filter: ["**/*.sketch.md"]
    combine: merge
    config:
      MD018: false # no space after # on a heading
      MD041: false # first line isn't a heading
```

## Contracts

`contracts/ticket`, `contracts/card`, `contracts/timestamp` and `contracts/validation` are public. Other tools, such as [Railyard](https://github.com/theroymind/railyard), import them to read tix's output.

## Development

```bash
make hooks   # point git at .githooks
make check   # format, tidy, build, vet, test, lint
make build   # build every app into bin/
```
