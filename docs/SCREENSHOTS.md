# Screenshot Checklist

This file tracks the GitHub-facing screenshot set. Keep it honest: only list a
screen as captured when the image exists under `docs/assets/screenshots/` and
matches the current TUI.

## Captured

| Screen | File | Purpose |
| --- | --- | --- |
| Home | `docs/assets/screenshots/home.png` | Shows primary workflows and current shell framing |
| Analyze | `docs/assets/screenshots/analyze.png` | Shows disk analysis and staged review handoff |
| Review | `docs/assets/screenshots/review.png` | Shows planned cleanup, protected findings, and execution control |

## Missing Before Public Release

| Screen | Placeholder name | Why it matters |
| --- | --- | --- |
| Permissions | `docs/assets/screenshots/permissions.png` | Demonstrates admin/dialog/native handoff preflight before execution |
| Progress | `docs/assets/screenshots/progress.png` | Demonstrates the explicit `Progress`, `Meter`, `Phase`, `Current`, `Next`, and `Status` contract |
| Result | `docs/assets/screenshots/result.png` | Demonstrates final outcome, skipped/protected items, and follow-up actions |

## Capture Instructions

Use deterministic fixture data and reduced motion so screenshots are stable:

```bash
go build -o ./sift ./cmd/sift
SIFT_REDUCED_MOTION=1 ./hack/capture_readme_screens.sh
```

If the capture helper does not yet produce a required screen, capture manually
from the same fixture roots and keep the terminal size consistent with the
existing images. Do not capture against personal directories or real user data.

Before committing refreshed screenshots:

- Confirm the image names match the tables above.
- Confirm no username, home path, machine name, token, or private app name is visible.
- Confirm Progress includes a clear percentage or meter.
- Confirm Permissions shows preflight requirements without implying automatic approval.
- Confirm Result shows final state and follow-up work without overstating cleanup success.
