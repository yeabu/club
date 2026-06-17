---
name: club-project
description: Club 阅卷与学情平台项目约定。Use when working in the club project, continuing implementation, or when the user says “更新仓库”; in that case review changes, run reasonable checks, create a git commit, and push to yeabu/club when remote access is configured.
---

# Club Project

## Project Defaults

- Treat `club/` as the engineering root.
- Keep the first-stage product focused on: paper templates, scan import, OCR/OMR grading, subjective-question review, score statistics, wrong-question archive, and learning analytics.
- Keep AI as an embedded capability layer, not as the primary product. AI may suggest regions, question types, answers, subjective scores, reasons, and wrong-cause analysis. Teachers retain final grading authority.
- Use a lightweight modern SaaS style for the Web teacher surface.
- Prioritize Go for the main backend API and Python for OCR/AI worker services.
- Keep Android configured first for the mobile student/guardian surface.

## Repository Update Trigger

When the user says `更新仓库`:

1. Inspect `git status --short`.
2. Review the relevant diff for files changed in this session.
3. Run reasonable available checks for touched areas.
4. Stage intentional project changes.
5. Commit with a concise Chinese commit message.
6. Push the current branch to the GitHub repository `yeabu/club` if a remote and credentials are available.

Do not commit unrelated user changes unless they are required for the requested update.

## Current Prototype Direction

- Web teacher dashboard uses the “阅卷工作台型” concept.
- The dashboard must include subjective-question grading as a core workflow.
- Subjective grading uses left/right split view: standard answer and scoring rules on the left, student paper/OCR answer and AI score suggestion on the right, with teacher final score controls.

