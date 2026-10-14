"""Verify requested historical Git metadata in parent order, not date order."""
import datetime as dt
import json
import pathlib
import subprocess

schedule = json.loads(pathlib.Path("docs/commit-schedule.json").read_text(encoding="utf-8"))
shas = subprocess.check_output(["git", "rev-list", "--reverse", "--first-parent", "HEAD"], text=True).splitlines()
assert len(shas) == len(schedule["commits"]), (len(shas), len(schedule["commits"]))
rows = []
previous = None
for sha, planned in zip(shas, schedule["commits"]):
    values = subprocess.check_output(["git", "show", "--no-patch", "--format=%H%n%s%n%aI%n%cI", sha], text=True).splitlines()
    actual_sha, message, author, committer = values
    stamp = dt.datetime.fromisoformat(author)
    assert author == committer == planned["date"], values
    assert message == planned["message"], values
    assert stamp.utcoffset() == dt.timedelta(hours=3), values
    assert dt.date(2026, 7, 10) <= stamp.date() <= dt.date(2026, 10, 14), values
    assert previous is None or stamp > previous, values
    previous = stamp
    rows.append({"sha": actual_sha, "message": message, "AuthorDate": author, "CommitDate": committer})
path = pathlib.Path("artifacts/commit-history.json")
path.parent.mkdir(exist_ok=True)
path.write_text(json.dumps(rows, indent=2) + "\n", encoding="utf-8")
print(json.dumps(rows, indent=2))
