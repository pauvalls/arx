# Deprecation Policy

**Applies to:** arx v1.0.0 and later.

## Summary

arx follows a **minimum 3-release deprecation window**. When a feature, field, or behavior is deprecated, it remains functional for at least 3 minor releases before removal. This gives users time to migrate without breakage.

## Policy

### 1. Deprecation notice

When a feature is deprecated:

1. **Documentation is updated** — The field or feature is marked with `[deprecated]` in the config reference and CLI reference docs.
2. **`arx check` warns** — Using a deprecated config field prints a warning to stderr:
   ```
   ⚠️  Deprecated: field "max_violations" is deprecated since v1.0.
       Use "severity_config" instead. Will be removed in v2.0.
   ```
3. **Deprecation header in JSON output** — The JSON output includes a `deprecations` field listing active deprecations.
4. **No exit code change** — Deprecation warnings never change the exit code (always 0).

### 2. Minimum window

- A deprecated feature is **removed no earlier than 3 minor releases** after deprecation.
- Example: deprecated in v1.1 → earliest removal in v1.4.
- Major version bumps (v1 → v2) reset the counter and MAY remove all deprecated features at once.

### 3. Migration path

Every deprecation MUST include:

- A replacement: what to use instead.
- A migration command: `arx config migrate` when applicable.
- Documentation: the config reference and CLI reference explain the migration.

### 4. Exceptions

- **Security fixes**: A feature MAY be removed immediately if it poses a security risk.
- **Bugs**: Features that cannot work correctly MAY be removed with a single-release notice.
- Both exceptions require a CHANGELOG entry explaining why the normal policy is bypassed.

## Example timeline

| Release | Event | Status |
|---------|-------|--------|
| v1.0 | Field `max_violations` is stable | ✅ Active |
| v1.1 | `max_violations` deprecated in favor of `severity_config.fail_build` | ⚠️ Warning appears |
| v1.2 | Still works, still warns | ⚠️ Warning |
| v1.3 | Still works, still warns | ⚠️ Warning |
| v1.4 | Removed. `arx config migrate` upgrades to `severity_config` | ❌ Removed |

## What this means for you

- **Upgrade within 3 releases** and you'll never see a broken config.
- **Run `arx config validate`** after every upgrade to see deprecation warnings.
- **Run `arx config migrate`** to auto-upgrade your config to the latest schema.
