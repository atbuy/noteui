# Docs maintenance

This page is for contributors working on the documentation site itself.

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

The workflow:

1. checks out the repo
2. installs the pinned `zensical` version
3. runs `zensical build --clean`
4. uploads the generated `site/` directory as the Pages artifact
5. deploys that artifact with `actions/deploy-pages`

The workflow uses a run-attempt-specific artifact name so reruns do not collide with older `github-pages` artifacts from the same workflow run.

## Common contributor checks

- If a page is missing from the site, verify it is present in `zensical.toml` navigation.
- If rendering differs between local and CI, compare the `zensical` version first.
- If deployment succeeds but the live site still looks old, verify the latest workflow run is the active Pages deployment and rule out browser caching.
- If `deploy-pages` reports multiple `github-pages` artifacts, confirm the workflow is still passing the same unique artifact name to both upload and deploy.
