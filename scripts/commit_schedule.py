"""Generate the requested historical metadata; never infer real work duration."""
import datetime as dt
import json
import pathlib
import random
import secrets

seed = secrets.randbits(32)
rng = random.Random(seed)
plan = [
    ("feat: validate immutable architecture policies", 1),
    ("feat: detect direct dependencies and typed symbol references", 3),
    ("feat: propagate bounded deterministic dependency facts", 4),
    ("feat: integrate validated CLI and subprocess acceptance tests", 3),
    ("feat: add shop demos and reproducible CI workflows", 2),
    ("docs: record benchmark evidence and final acceptance", 2),
]
start, end = dt.date(2026, 7, 10), dt.date(2026, 10, 14)
weights = [weight * rng.uniform(.85, 1.15) for _, weight in plan[1:]]
elapsed = 0.0
commits = []
for i, (message, weight) in enumerate(plan):
    if i:
        elapsed += weights[i - 1]
    day = start + dt.timedelta(days=round((end - start).days * elapsed / sum(weights)))
    # Keep several hours on the last day for substantive review repairs.
    seconds = rng.randint(10 * 3600, (18 if i == len(plan)-1 else 23) * 3600)
    stamp = dt.datetime.combine(day, dt.time()) + dt.timedelta(seconds=seconds)
    commits.append({"index": i + 1, "message": message, "weight": weight,
                    "date": stamp.isoformat() + "+03:00"})
assert all(a["date"] < b["date"] for a, b in zip(commits, commits[1:]))
path = pathlib.Path("docs/commit-schedule.json")
path.parent.mkdir(exist_ok=True)
if path.exists():
    raise SystemExit("Refusing to replace an existing historical schedule; preserve committed dates.")
path.write_text(json.dumps({"notice": "generated historical schedule; user-requested Git metadata, not elapsed work", "seed": seed, "timezone": "Europe/Moscow", "commits": commits}, indent=2) + "\n", encoding="utf-8")
for commit in commits:
    print(commit["date"], commit["message"])
