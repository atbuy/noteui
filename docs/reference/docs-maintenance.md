# Docs maintenance

This page is for contributors working on the documentation site itself.

## Contributor setup

From a git clone, run this once before you start editing:

```bash
make tools
```

That installs the shared Go formatter and lint tools, installs `pre-commit` via `uv`, and registers the repository hooks used by contributors.

## Local docs build

Build the docs locally from the repository root:

```bash
uvx zensical build --clean
```

Serve locally during editing:

```bash
uvx zensical serve
```

The docs source lives under `docs/` and the generated site is written to `site/`.

## Tooling version

The GitHub Pages workflow pins:

```text
zensical==0.0.31
```

Keep local verification aligned with that version when diagnosing rendering differences between local output and GitHub Pages.

## Site configuration

The docs navigation, theme, markdown extensions, and site URL live in:

```text
zensical.toml
```

When you add a new page, update the nav there so the page is included in the built site.

## GitHub Pages deployment

GitHub Pages is deployed by the `Documentation` workflow in `.github/workflows/docs.yml`.

The workflow uses the standard two-job GitHub Pages pattern:

1. the `build` job checks out the repo
2. configures Pages metadata
3. installs the pinned `zensical` version
4. runs `zensical build --clean`
5. uploads the generated `site/` directory as the Pages artifact
6. the `deploy` job publishes that artifact with `actions/deploy-pages`

## Common contributor checks

- Run `make check` before sending a larger change. It covers lint, unit tests, race tests, and a full build.
- If you are only touching docs, `make tools` plus `make docs-build` is usually the minimum local verification flow.
- If a page is missing from the site, verify it is present in `zensical.toml` navigation.
- If rendering differs between local and CI, compare the `zensical` version first.
- If deployment succeeds but the live site still looks old, verify the latest workflow run is the active Pages deployment and rule out browser caching.
- If Pages deployment fails, verify the workflow is still using the standard separate `build` and `deploy` jobs and that `site/` is uploaded by `actions/upload-pages-artifact`.
