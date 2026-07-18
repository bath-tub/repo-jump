# repo-jump

Fast terminal fuzzy-finder that opens a GitHub repo in your browser. Type part
of a repo name, hit enter, and it opens `https://github.com/<org>/<name>` in a
new tab. Ranking blends fuzzy-match quality with a self-learning **frecency**
signal, so the repos you actually use float to the top — no list to maintain,
and it works identically for anyone (the only state is your own usage).

## Install

One command — builds the `rj` binary, puts it on your PATH, then runs the
interactive setup wizard:

```sh
git clone https://github.com/bath-tub/repo-jump.git
cd repo-jump && ./install.sh
```

The wizard checks the [`gh`](https://cli.github.com) CLI (offering to run
`gh auth login` if needed), asks which org/owner to jump within, builds the repo
index, and optionally adds a Ctrl-G zsh keybinding. Re-run it anytime with
`rj setup`.

Requires [Go](https://go.dev/dl) (build) and [`gh`](https://cli.github.com)
(indexing). Install location defaults to `~/.local/bin`; override with
`REPO_JUMP_BIN=/somewhere ./install.sh`.

## Use

```sh
rj            # or press Ctrl-G if you added the keybinding
```

- Type — subsequence fuzzy match (`kc` → `kube_config`), matched chars highlighted.
- `↑/↓` (or `ctrl-p/ctrl-n`) to move, `enter` to open, `esc` to quit.
- Empty query shows your most-used repos first (★ marks ones you've opened).

Refresh the index whenever repos are added:

```sh
rj --refresh                 # re-index the saved org
rj --refresh --org other-org # switch to a different org (also saved)
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
`repos.txt` (the index) and `frecency.json` (your usage). The chosen org is
saved in `$XDG_CONFIG_HOME/repo-jump/org` (or `~/.config/repo-jump/org`).
