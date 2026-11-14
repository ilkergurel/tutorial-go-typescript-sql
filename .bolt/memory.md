# Project memory

## Product intent & audience

- Local-only, full-stack learning platform teaching programming through runnable code, interactive lessons, and architectural guides.
- Audience: programming learners using a self-hosted/local environment.

## Brand personality & voice

- Technical, educational, production-quality. Favors clear checkpoints and verifiable progress over marketing language.

## Visual design & UX direction

- Vite + React + TypeScript + Tailwind CSS + Lucide React icons (per `.bolt/prompt`).
- `@/` path alias maps to `src/`.
- Lesson sidebar shows status dots; dashboard cards show progress lines; lesson headers show status badges and bookmark buttons.
- Lesson pages use tabs (e.g. content, quiz, notes). Notes editor auto-saves with confirmation feedback.
- Quiz UI locks answers on selection, reveals explanations once, shows running score.

## Decisions, constraints & rationale

- Backend: local Go server + PostgreSQL (`app-db`); persistence governed by ADR-0002 and ERD in `ARCHITECTURE.md`.
- Progress states move forward only: viewed → ran → completed; never downgrade.
  - "viewed" set automatically on lesson visit; "ran" on successful code run; "completed" by user mark or perfect quiz.
- Offline fallback: if backend unreachable (frontend standalone), silently uses browser storage so features still work.
- New DB migration covers progress, notes, bookmarks, and quiz-answer tables, each with a backend endpoint.
- Development is phase-based; Phase 4 (learning features) shipped. Phase 5 is content batches.
- Verification gate before stopping: lint, typecheck, frontend build, Go build/vet, runner integration tests.
- Docker stack and live UI interaction cannot be exercised in sandbox; require manual verification on user's machine.

## Open questions & assumptions

- Fixture files (e.g. reading-files example) don't survive version restores; may need rebuilding after restores.
- Phase 5 scope details (content volume, topics) not yet specified beyond "content batches."
- Docker stack assumed working but unverified in sandbox.