# Encrypted notes

noteui supports encrypted note bodies for users who want notes stored in encrypted form on disk while still being able to preview and edit them through the app.

## How it works

- the note body is encrypted
- the encrypted state is reflected in frontmatter
- noteui can decrypt for preview or editing workflows when needed
- edited content can be re-encrypted when written back

## What this is good for

- personal notes that you want to keep encrypted on disk
- workflows where file portability still matters
- users who want encryption without adopting a separate note storage format

## Important constraints

!!! warning

    This is an application workflow for encrypted note bodies. It is not a general-purpose secret-management system.

## Frontmatter signal

Encrypted notes use the `encrypted` frontmatter field.

Example:

```yaml
---
encrypted: true
---
```

## Related workflows

- [Usage guide](../guide/usage.md)
- [Storage and state](../reference/storage-and-state.md)
