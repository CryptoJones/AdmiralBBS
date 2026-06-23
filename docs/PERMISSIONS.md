# Permissions — AdmiralBBS

> Access is gated by a single integer `access_level` on each `user`, compared
> against a `min_access_level` on areas/doors. Higher = more privileged.

## Access levels

| Level | Role | Can |
|---|---|---|
| 100 | **SysOp** | Everything: approve/deny members, manage areas, run SysOp tools, read the audit log |
| 80 | **Co-SysOp** | Moderate boards, manage files, approve members |
| 50 | **Member** | Full caller access to areas at/below their level (boards, mail, files, doors) |
| 10 | **Guest** | Read-only / limited access; cannot post or download unless an area allows it |
| 0 | **Pending** | Applied but not yet approved — login allowed only to a holding screen |

## Rules

- A `pending` user (membership not approved) is held at an application/holding
  screen and cannot enter subsystems. (Approval workflow = open question in
  `planning/QUESTIONS.md`.)
- Every message area, file area, and door carries a `min_access_level`; the
  menu engine hides what a caller cannot reach.
- The daily time budget (`daily_minutes`) applies to all non-SysOp levels.
- The audit log (`session_log`) is SysOp-only.

---

*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
