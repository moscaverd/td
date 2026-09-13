#!/usr/bin/env python3
"""Exercise a built CLI using disposable projects and configuration only."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile


binary = Path(sys.argv[1]).resolve(strict=True)
with tempfile.TemporaryDirectory(prefix="td-projects-smoke-") as temporary:
    fixture = Path(temporary).resolve()
    project = fixture / "project spaces#mode=rw%"
    project.mkdir()
    config = fixture / "config"
    environment = dict(os.environ)
    environment.update(
        TD_CONFIG_DIR=str(config),
        TD_FEATURE_SYNC_AUTOSYNC="0",
        TD_FEATURE_SYNC_CLI="0",
        TD_LOG_FILE=str(fixture / "fixture.log"),
    )

    def run(*arguments, cwd=project):
        result = subprocess.run(
            [str(binary), *arguments],
            cwd=cwd,
            env=environment,
            input="n\n",
            text=True,
            capture_output=True,
            timeout=15,
            check=True,
        )
        return result.stdout

    run("init")
    run("create", "Isolated project aggregation fixture")
    assert str(project) in run("projects")
    assert str(project) in run("pj")
    assert "Isolated project aggregation fixture" in run("projects", "list")
    entries = json.loads((config / "projects.json").read_text())
    assert len(entries) == 1 and entries[0]["path"] == str(project)
    database = project / ".todos" / "issues.db"
    before = hashlib.sha256(database.read_bytes()).hexdigest()
    assert "Isolated project aggregation fixture" in run(
        "projects", "list", cwd=fixture
    )
    assert hashlib.sha256(database.read_bytes()).hexdigest() == before

print(json.dumps(dict(init=True, create=True, registry=True, aggregate=True,
                     alias=True, unrelated_cwd=True, database_unchanged=True)))
