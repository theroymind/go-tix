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
sketch                    # the explorer; enter/e edit · a add · d delete · r link · y ready · > answer · ]/[ questions · f feature · v plan · ? help
sketch ls --json          # every note with its target, status, body, links and questions, for agents
sketch questions --json   # the open questions; --all adds answered ones with their answers
sketch plan --feature X   # X's brief as markdown: tix criteria, a Mermaid graph, notes in build order, decisions
sketch graph --mermaid    # only the Mermaid flowchart, a subgraph per folder; takes --feature X too
```

A note is a blueprint for whoever builds the file, person or agent, and its whole body is read as written. sketch reads only a few lines, and `sketch help` has the details:

```markdown
---
feature: camera-controls
status: draft
---
# useOrbit.ts

#fn useOrbit(target: Ref<Vec3>) -> { yaw, pitch }
#test releasing the drag eases back to rest
-> ./useCamera.ts : reads the pitch limits

? clamp pitch here or in useCamera
> in useCamera,
> it owns the limits
```

`feature:` links the note to a tix feature, and `status:` is `draft` or `ready`. A `-> path : label` line links to another file, from the repo root or, starting with `./` or `../`, from the note's folder. A path holding a space, or a URL such as `-> https://example.com`, leaves the line free text, so an arrow starting a sentence is never a link. In the explorer, `r` picks the file and adds the line, and the line above the footer shows the note's `uses →` and `used by ←` links, `!` marking a path that is neither a file nor planned. A `?` line is an open question until `>` lines right under it, or under one blank line, answer it. `#fn`, `#type`, `#component`, `#flow` and `#test` are a suggested convention for describing what the file holds; sketch doesn't parse them. tix is optional: without it, `sketch ls` warns and skips checking features, and `sketch plan` leaves out the feature's criteria. `sketch plan` puts each note after every note it links to, and the notes of a cycle together in path order, and moves a note's own headings three levels down, so its `#` sits under the note's `###`.

In LazyVim, this spec in `~/.config/nvim/lua/plugins/sketch.lua` opens it as a left split on `<leader>k`, and `<leader>k` again hides it. Inside an nvim with the spec loaded, the explorer hides its preview pane: moving onto a note, in the tree or the plan view's list, shows it in the main window while focus stays in sketch, and `enter`, `e`, `a` or `>` move focus to it there, `>` on the answer line. sketch calls the spec's `sketch_show` over `nvim --remote-expr`; in an nvim without the spec, it keeps its preview pane and opens notes with `nvim --remote`. The explorer reloads when its terminal regains focus, and `R` reloads it by hand.

```lua
local width = 38
local term

local function toggle()
  term = Snacks.terminal.toggle(vim.g.sketch_cmd or "sketch", {
    cwd = LazyVim.root(),
    win = { position = "left", width = width },
  })
end

-- A window a note can show in: a normal, non-floating window that isn't sketch's.
local function is_main(win)
  return vim.api.nvim_win_is_valid(win)
    and not (term and win == term.win)
    and vim.api.nvim_win_get_config(win).relative == ""
    and vim.bo[vim.api.nvim_win_get_buf(win)].buftype == ""
end

-- The window notes show in: the one used last, else any main one, else a new one right of sketch.
local function main_window()
  local last = vim.fn.win_getid(vim.fn.winnr("#"))
  if is_main(last) then
    return last
  end
  for _, win in ipairs(vim.api.nvim_tabpage_list_wins(0)) do
    if is_main(win) then
      return win
    end
  end
  local win = vim.api.nvim_open_win(vim.api.nvim_create_buf(false, true), false, { split = "right", win = -1 })
  if term and term.win and vim.api.nvim_win_is_valid(term.win) then
    vim.api.nvim_win_set_width(term.win, width) -- the split above evened the widths out
  end
  return win
end

-- sketch calls this over nvim --remote-expr. It shows the note at path in the main window, moving focus there when
-- focus is set, with the cursor on line when it's above 0. A deleted note isn't opened, and a window holding
-- unsaved edits with 'nohidden' set keeps them and doesn't show the note.
function _G.sketch_show(path, focus, line)
  if vim.fn.filereadable(path) == 0 then
    return
  end
  local win = main_window()
  local ok = pcall(vim.api.nvim_win_call, win, function()
    vim.cmd.edit(vim.fn.fnameescape(path))
  end)
  if not ok then
    return
  end
  if line > 0 then
    local last = vim.api.nvim_buf_line_count(vim.api.nvim_win_get_buf(win))
    vim.api.nvim_win_set_cursor(win, { math.min(line, last), 0 })
  end
  if focus then
    vim.cmd.stopinsert()
    vim.api.nvim_set_current_win(win)
  end
end

return {
  "folke/snacks.nvim",
  keys = {
    { "<leader>k", toggle, desc = "Sketch explorer" },
  },
}
```

The `#fn`-style lines read as headings without a space to markdownlint, and a note needn't start with a heading. With markdownlint-cli2 0.23.0 or later, as LazyVim's markdown extra runs, this `.markdownlint-cli2.yaml` at the repo root turns off those two rules for notes only:

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
