---
tags: project, active
---

# Project Alpha

## Goals

- Ship the v1.0 release by end of quarter
- Achieve 95% test coverage across all packages
- Write user-facing documentation for every feature

## Current status

The core feature set is complete. Remaining work is polish: edge cases in the
sync protocol, a handful of UI layout issues at narrow terminal widths, and
filling gaps in the docs.

## Next steps

1. Fix the layout regression at widths below 80 columns
2. Write the sync troubleshooting guide
3. Tag the release and publish the archive

## Notes

The sync binary needs to be co-versioned with the main binary. Pin both to the
same release tag to avoid protocol mismatches.
