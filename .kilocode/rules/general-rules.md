# general-rules.md
This is General rules for the AI to works, thinks and provide solution
this rules has to be followed diligiently 

# Guidelines

## Purpose
Ensure solutions are correct, simple, and aligned with established best practices.
Normative keywords: MUST, SHOULD, MAY follow [RFC 2119] semantics.

### Core Principles
- Solutions MUST meet the stated requirement with the simplest viable design.
- Code MUST follow the language’s accepted best practices and style guide.
- Avoid over-engineering: prefer straightforward approaches over patterns/frameworks unless they are clearly justified by requirements.
- Choose readability and maintainability first; optimize only for real constraints (performance, memory, security, availability).
### Style & Standards
- When a project toolchain is specified, use it. If none is specified, default to:
- Go: gofmt, go vet, golangci-lint; follow Effective Go; follow Go sonarqube way as possible .

- PHP: PSR-12, phpcs/php-cs-fixer; prefer strict types.

- Java: Google Java Style, spotbugs/checkstyle.

- Python: PEP 8, black, ruff, type hints (PEP 484).

- JS/TS: ESLint + default recommended rules, Prettier.

- Delphi/Pascal: Consistent naming, one unit = one responsibility, PascalCase types, CamelCase methods, meaningful unit names.

### General style rules
- Functions SHOULD be ≤ ~30–50 lines; types/methods SHOULD have single, clear responsibility.
- Public APIs MUST be documented (purpose, params, returns, errors).
- Names MUST be descriptive; avoid abbreviations unless domain-standard.

### Simplicity Guardrails (Anti Over-Engineering)
- Do not introduce frameworks, patterns, or abstractions unless they reduce current duplication or meet a stated requirement.
- Do not add configuration flags, extension points, or generic types “just in case.”
- Prefer standard library; add dependencies only when a clear benefit outweighs maintenance risk.
- Cyclomatic complexity SHOULD be low; refactor deeply nested branches into smaller functions.
#### Examples Of Simplicity Guardrails
- ✅ OK: direct query + small mapper instead of introducing a Repository & Unit-of-Work for one read.
- ❌ Not OK: event bus + CQRS for a single in-process command with no scaling/consistency need.
- ✅ OK: slice + loop;
- ❌ Not OK: custom iterator/generics for 3 items.

### Correctness, Tests, and Examples
- New/changed logic MUST include minimal tests demonstrating correctness and edge cases.
- Include at least one runnable usage example (or a quickstart snippet).
- Validate inputs early; fail fast with clear error messages.
- Pure functions SHOULD be favored; side effects isolated.
- use Qdrant mcp as much as possilbe when you need to understand code within a file 
- after all task done then reindex the code base with Qdrant MCP tools

### Error Handling & Logging
- MUST return/propagate errors with actionable context (no silent failures).
- Logging SHOULD be structured (level, code, message, key fields).
- Do not log secrets or sensitive PII.

### Performance & Resource Use
- Respect obvious budgets (e.g., response < 100 ms, memory proportional to input).
- Prefer O(n) over O(n log n) where feasible; avoid unnecessary allocations and I/O.
- Measure before micro-optimizing; include notes if constraints drove a design choice.
### Security & Reliability
- MUST follow baseline security for the stack: parameterized queries, input validation, output encoding, least privilege, safe defaults.
- Handle timeouts, retries (with backoff), and cancellation where I/O is involved.
- Prefer idempotent operations for networked calls when possible.

### Dependencies & Architecture
- New dependencies MUST be: well-maintained, permissively licensed, version-pinned, and minimal.
- Public surface area SHOULD be small; internal details hidden.
- Code MUST be organized by domain/responsibility, not by technical layer only.

### Deliverable Format (for AI Outputs)
- Every code answer MUST include:
    - Short Summary (3–5 bullets): what was built and why.
    - Code Block(s): complete, runnable, minimal.
    - Usage Example: how to call/run it.
        - Tests (or test snippet): show success and at least one edge case.
        - Assumptions & Limits: list any trade-offs or TBDs.
- If requirements are ambiguous, output a brief “Questions to Clarify” list before finalizing the solution.

### Acceptance Checklist
- A submission is acceptable only if all are YES:
    -  Meets requirements without unnecessary features.
    -  Follows the specified (or default) style/linting rules.
    -  Adds only essential dependencies.
    -  Has tests and a usage example.
    -  Handles errors and avoids leaking sensitive data.
    -  Notes trade-offs and constraints.