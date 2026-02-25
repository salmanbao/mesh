---
name: mesh-dev-documentation
description: Write and update developer-facing documentation for mesh microservices so engineers can understand implementation details and reason about architectural and design decisions. Use this skill when documenting service behavior, boundaries, tradeoffs, and decision rationale in mesh code and docs.
---

# Mesh Dev Documentation

Use this skill to document mesh services for developers with clear behavior, boundaries, and decision rationale.

## Load First
- `references/documentation-standards.md`
- `references/documentation-workflow.md`
- `references/decision-rationale-template.md`

## Focus Areas
- Service-level documentation in `mesh/services/*/*/README.md`.
- Runtime and platform docs in `mesh/docs/`.
- Automation and operator docs in `mesh/scripts/README.md`.
- Contract reasoning and dependency notes linked to canonical specs.
- Why a design was chosen, what was rejected, and consequences.

## Workflow
1. Read the code path first (handlers, use-cases, ports, adapters, contracts).
2. Extract what the code does, not what it was intended to do.
3. Document architecture boundaries and dependency direction.
4. Record decision rationale and tradeoffs using the template.
5. Add operational and testing implications.
6. Validate docs are consistent with canonical spec sources and current script behavior.
7. For automation docs, include exact command examples, arguments, and failure modes.

## Non-Negotiables
- Keep documentation accurate to current code.
- Explain "why" and "tradeoffs", not only "what".
- Reference canonical sources when describing ownership/contracts.
- Avoid vague wording and marketing language.
- Do not document filename-only bootstrap assumptions; use package-level bootstrap contract (`Build` or `NewRuntime`).
