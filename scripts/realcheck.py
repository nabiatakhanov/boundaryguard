#!/usr/bin/env python3
"""Record a read-only BoundaryGuard run against an existing local Go repository."""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time


def timestamp() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat()


def run(command: list[str], directory: Path, environment: dict[str, str]) -> dict:
    started = time.perf_counter()
    completed = subprocess.run(
        command, cwd=directory, env=environment, capture_output=True, check=False
    )
    return {
        "command": command,
        "cwd": str(directory),
        "exit_code": completed.returncode,
        "wall_seconds": time.perf_counter() - started,
        "stdout": completed.stdout.decode("utf-8", errors="replace"),
        "stderr": completed.stderr.decode("utf-8", errors="replace"),
    }


def walk_error(error: OSError) -> None:
    raise error


def snapshot(directory: Path) -> dict:
    files = []
    for parent, directories, names in os.walk(directory, onerror=walk_error):
        # Git internals are not target sources; never refresh or inspect its index.
        directories[:] = [name for name in directories if name != ".git"]
        for name in names:
            if not (name.endswith(".go") or name in {"go.mod", "go.sum"}):
                continue
            path = Path(parent) / name
            digest = hashlib.sha256()
            size = 0
            with path.open("rb") as source:
                for chunk in iter(lambda: source.read(1024 * 1024), b""):
                    digest.update(chunk)
                    size += len(chunk)
            files.append(
                {
                    "path": path.relative_to(directory).as_posix(),
                    "sha256": digest.hexdigest(),
                    "bytes": size,
                }
            )
    files.sort(key=lambda item: item["path"])
    tree = hashlib.sha256()
    for entry in files:
        name = entry["path"].encode("utf-8")
        tree.update(len(name).to_bytes(8, "big"))
        tree.update(name)
        tree.update(bytes.fromhex(entry["sha256"]))
    return {
        "sha256": tree.hexdigest(),
        "file_count": len(files),
        "total_bytes": sum(entry["bytes"] for entry in files),
        "files": files,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--config", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True, help="JSON evidence file outside target repository")
    args = parser.parse_args()
    repository = args.repo.resolve(strict=True)
    binary = args.binary.resolve(strict=True)
    policy = args.config.resolve(strict=True)
    output = args.out.resolve()
    if not repository.is_dir() or not binary.is_file() or not policy.is_file():
        parser.error("--repo must be a directory; --binary and --config must be files")
    if output == repository or repository in output.parents:
        parser.error("--out must be outside the target repository")

    environment = dict(os.environ)
    environment.update(
        GOWORK="off", GOPROXY="off", GOFLAGS="-mod=readonly", GIT_OPTIONAL_LOCKS="0"
    )
    evidence = {
        "started_at_utc": timestamp(),
        "repository": str(repository),
        "binary": str(binary),
        "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
        "config": str(policy),
        "config_sha256": hashlib.sha256(policy.read_bytes()).hexdigest(),
        "policy": json.loads(policy.read_text(encoding="utf-8")),
        "environment_overrides": {
            key: environment[key]
            for key in ("GOWORK", "GOPROXY", "GOFLAGS", "GIT_OPTIONAL_LOCKS")
        },
        "python_version": sys.version,
        "source_selection": "production packages ./...; -test=false; active platform/build constraints",
        "completed": False,
    }
    exit_code = 1
    status_command = ["git", "--no-optional-locks", "-C", str(repository), "status", "--short"]
    try:
        evidence["go_version"] = run(["go", "version"], repository, environment)
        evidence["git_version"] = run(["git", "--version"], repository, environment)
        evidence["before"] = snapshot(repository)
        evidence["git_before"] = run(status_command, repository, environment)
        if evidence["git_before"]["exit_code"] != 0:
            raise RuntimeError("cannot capture initial Git status; see git_before")
        try:
            evidence["analysis"] = run(
                [str(binary), "-config", str(policy), "-test=false", "./..."],
                repository,
                environment,
            )
        finally:
            evidence["after"] = snapshot(repository)
            evidence["git_after"] = run(status_command, repository, environment)
        evidence["sources_unchanged"] = evidence["before"] == evidence["after"]
        evidence["git_status_unchanged"] = (
            evidence["git_after"]["exit_code"] == 0
            and evidence["git_before"]["stdout"] == evidence["git_after"]["stdout"]
        )
        if not evidence["sources_unchanged"] or not evidence["git_status_unchanged"]:
            raise RuntimeError("target tree or Git status changed; inspect before/after evidence")
        analysis_code = evidence["analysis"]["exit_code"]
        if analysis_code not in (0, 3):
            raise RuntimeError(f"analyzer failed with unexpected exit code {analysis_code}")
        evidence["analysis_result"] = "clean for selected policy" if analysis_code == 0 else "policy diagnostics"
        evidence["completed"] = True
        exit_code = 0
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        evidence["error"] = str(error)
    finally:
        evidence["finished_at_utc"] = timestamp()
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"Evidence: {output}")
    if exit_code:
        print(evidence.get("error", "experiment did not complete"), file=sys.stderr)
    else:
        print(f"Analyzer exit {evidence['analysis']['exit_code']}; selected sources and Git status unchanged")
    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())
