# repo-jump

Fast terminal fuzzy-finder that opens a GitHub repo in your browser. Type part
of a repo name, hit enter, and it opens `https://github.com/<org>/<name>` in a
new tab. Ranking blends fuzzy-match quality with a self-learning **frecency**
signal, so the repos you actually use float to the top — no list to maintain,
and it works identically for anyone (the only state is your own usage).

## Install

```sh
go build -o repo-jump .
# then put ./repo-jump on your PATH
```

## Setup

Build the local repo index once (repeat whenever repos are added):

```sh
repo-jump --refresh --org my-org    # an org/owner you can see via gh
repo-jump --refresh                 # no --org: defaults to your gh account
```

The chosen org is saved, so later runs need no `--org`. This requires the
[`gh`](https://cli.github.com) CLI, authenticated against an account that can
see the org's repos (`gh auth status`).

## Use

```sh
repo-jump
```

- Type — subsequence fuzzy match (`kc` → `kube_config`), matched chars highlighted.
- `↑/↓` (or `ctrl-p/ctrl-n`) to move, `enter` to open, `esc` to quit.
- Empty query shows your most-used repos first (★ marks ones you've opened).

Bind it to a keystroke in your shell for instant access, e.g. zsh:

```sh
bindkey -s '^g' 'repo-jump\n'   # ctrl-g from an empty prompt
```

## Configuration

| Flag / env | Default | Meaning |
|---|---|---|
| `--org` / `REPO_JUMP_ORG` | saved org, else your gh account | GitHub org/owner to jump within |
| `--alpha` / `REPO_JUMP_ALPHA` | `2.0` | weight applied to the frecency signal |
| `--refresh` | — | rebuild the repo index via `gh repo list` and exit |

## How ranking works

For each candidate: `finalScore = fuzzyScore + alpha · log2(1 + frecency)`.

- **fuzzyScore** rewards contiguous runs and word-boundary hits, and lightly
  prefers shorter names.
- **frecency** is `visitCount × recencyMultiplier` (zoxide-style buckets: ×4
  within an hour, ×2 a day, ×0.5 a week, ×0.25 older), dampened by `log2` so a
  hot repo boosts strongly without bulldozing a clearly-better textual match.

State lives in `$XDG_DATA_HOME/repo-jump/` (or `~/.local/share/repo-jump/`):
`repos.txt` (the index) and `frecency.json` (your usage).
