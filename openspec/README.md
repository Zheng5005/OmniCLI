# OpenSpec — OmniCLI

This directory contains the **Spec-Driven Development** (SDD) artifacts for OmniCLI.

## Layout

```
openspec/
├── README.md                 ← this file
├── specs/                    ← source of truth (current behavior)
│   └── omnicli-v1/spec.md
└── changes/                  ← active changes + archive
    └── archive/              ← completed changes (audit trail)
        └── 2026-04-25-omnicli-v1/
            ├── proposal.md
            ├── spec.md
            ├── design.md
            ├── tasks.md
            ├── apply-progress.md
            ├── verify-report.md
            └── archive-report.md
```

## Workflow

1. **Propose** — write a `proposal.md` with intent, scope, approach, risks
2. **Spec** — define requirements + scenarios in `spec.md` (Given/When/Then, RFC 2119 keywords)
3. **Design** — document architectural decisions in `design.md`
4. **Tasks** — break the work into a checklist in `tasks.md`
5. **Apply** — implement; track in `apply-progress.md`
6. **Verify** — run tests, build, type checks; produce `verify-report.md`
7. **Archive** — sync delta specs into `specs/` and move change folder to `archive/`

## Conventions

- Use `Given/When/Then` for scenarios
- Use RFC 2119 keywords (MUST, SHALL, SHOULD, MAY) for requirements
- Archive folders are prefixed with `YYYY-MM-DD-{change-name}`
- Never modify archived changes — they are the immutable audit trail

## Cycle Status

| Change | Status | Verdict |
|--------|--------|---------|
| `omnicli-v1` | ✅ Archived (2026-04-25) | PASS WITH WARNINGS |
