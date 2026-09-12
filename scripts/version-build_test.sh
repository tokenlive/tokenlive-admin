#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

assert_contains() {
  [[ "$1" == *"$2"* ]] || { echo "missing build argument: $2" >&2; exit 1; }
}

output="$(RELEASE_TAG=v9.8.7 BUILD_KIND=release make -n build-frontend build)"
assert_contains "$output" "VITE_APP_VERSION=v9.8.7"
assert_contains "$output" "main.VERSION=v9.8.7"
assert_contains "$output" "main.BUILD_KIND=release"

for target in start build build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64; do
  output="$(RELEASE_TAG=v9.8.7 BUILD_KIND=release make -n "$target")"
  assert_contains "$output" "main.VERSION=v9.8.7"
  assert_contains "$output" "main.BUILD_KIND=release"
  output="$(env -u BUILD_KIND RELEASE_TAG=v9.8.7 make -n "$target")"
  assert_contains "$output" "main.BUILD_KIND=dev"
done

output="$(env -u RELEASE_TAG -u BUILD_KIND make -n build)"
assert_contains "$output" "main.BUILD_KIND=dev"

for target in docker-build docker-push; do
  output="$(RELEASE_TAG=v9.8.7 BUILD_KIND=release make -n "$target")"
  assert_contains "$output" "--build-arg VERSION=v9.8.7"
  assert_contains "$output" "--build-arg RELEASE_TAG=v9.8.7"
  assert_contains "$output" "--build-arg BUILD_KIND=release"
done

if RELEASE_TAG=latest make -n build >/dev/null 2>&1; then
  echo "latest was accepted as a runtime version" >&2
  exit 1
fi

# Execute actual build commands extracted from Dockerfiles/workflows in
# disposable directories. Only compilers are fake; no CI, image or deployment
# operation is dispatched.
python3 - <<'PY'
import json
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile

repo = pathlib.Path.cwd()
fake_tool = """
import json, os, pathlib, sys
tool = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
if tool == "git":
    if args == ["rev-parse", "--short", "HEAD"]:
        print("0123456")
        sys.exit(0)
    sys.exit("unexpected git operation")
if tool == "npm" and args not in (["ci"], ["install"], ["run", "build:prod"]):
    sys.exit("unexpected npm command")
with open(os.environ["COMMAND_LOG"], "a") as log:
    log.write(json.dumps({"tool": tool, "args": args,
        "version": os.environ.get("VITE_APP_VERSION")}) + "\\n")
"""

def fixture(root):
    tools = root / "tools"
    tools.mkdir()
    for tool in ("go", "npm", "git"):
        path = tools / tool
        path.write_text(f"#!{sys.executable}\n" + fake_tool)
        path.chmod(0o755)
    (root / "frontend").mkdir()
    shutil.copyfile(repo / "Makefile", root / "Makefile")
    return {
        "PATH": f"{tools}:/usr/bin:/bin",
        "COMMAND_LOG": str(root / "commands.jsonl"),
    }

def entries(root):
    log = root / "commands.jsonl"
    return [json.loads(line) for line in log.read_text().splitlines()] if log.exists() else []

def run(command, root, env):
    return subprocess.run(
        ["bash", "-e", "-c", command], cwd=root, env=env, text=True, capture_output=True
    )

def assert_metadata(root, version, kind, frontend=True, backend=True):
    calls = entries(root)
    if frontend:
        builds = [c for c in calls if c["tool"] == "npm" and c["args"] == ["run", "build:prod"]]
        assert len(builds) == 1, calls
        assert builds[0]["version"] == version, builds
    if backend:
        builds = [c for c in calls if c["tool"] == "go"]
        assert len(builds) == 1, calls
        args = builds[0]["args"]
        flags = args[args.index("-ldflags") + 1]
        assert f"main.VERSION={version}" in flags, args
        assert f"main.BUILD_KIND={kind}" in flags, args

for dockerfile in ("deploy/build/Dockerfile", "deploy/build/CN.Dockerfile"):
    instructions = (repo / dockerfile).read_text().replace("\\\n", " ").splitlines()
    for overrides, version, kind in (
        ({}, "dev", "dev"),
        ({"VERSION": "v9.8.7", "BUILD_KIND": "release"}, "v9.8.7", "release"),
        ({"VERSION": "v1.0.0", "RELEASE_TAG": "v9.8.7", "BUILD_KIND": "release"}, "v9.8.7", "release"),
        ({"RELEASE_TAG": "v9.8.7"}, "v9.8.7", "dev"),
        ({"VERSION": "latest"}, None, None),
    ):
        with tempfile.TemporaryDirectory(prefix="admin-docker-version-test-") as temporary:
            root = pathlib.Path(temporary)
            base_env = fixture(root)
            env = dict(base_env)
            results = []
            for line in instructions:
                if line.startswith("FROM "):
                    env = dict(base_env)
                if line.startswith(("ARG ", "ENV ")):
                    instruction, _, expression = line.partition(" ")
                    key, _, value = expression.partition("=")
                    value = re.sub(r"\$\{(\w+)\}", lambda m: env.get(m[1], ""), value)
                    env[key] = overrides.get(key, value) if instruction == "ARG" else value
                if line.startswith("RUN ") and ("go build " in line or "npm run build:prod" in line):
                    results.append(run(line[4:], root, env))
            assert len(results) == 2, dockerfile
            if version is None:
                assert all(result.returncode != 0 for result in results), f"{dockerfile} accepted latest"
                assert not entries(root), entries(root)
            else:
                assert all(result.returncode == 0 for result in results), results
                assert_metadata(root, version, kind)

def step(path, name):
    lines = path.read_text().splitlines()
    start = next(i for i, line in enumerate(lines) if line.strip() == f"- name: {name}")
    indent = len(lines[start]) - len(lines[start].lstrip())
    end = next(
        (i for i in range(start + 1, len(lines)) if lines[i].startswith(" " * indent + "- ")),
        len(lines),
    )
    lines = lines[start:end]
    run_line = next(i for i, line in enumerate(lines) if line.strip().startswith("run:"))
    inline = lines[run_line].strip()[4:].strip()
    command = (
        "\n".join(line[indent + 4:] for line in lines[run_line + 1:])
        if inline == "|" else inline
    )
    env = {}
    for line in lines[:run_line]:
        if line.startswith(" " * (indent + 4)) and ":" in line:
            key, value = line.strip().split(":", 1)
            env[key] = value.strip()
    return env, command

def resolve(text, values):
    return re.sub(r"\$\{\{\s*(.*?)\s*\}\}", lambda match: values[match[1]], text)

release = repo / ".github/workflows/release.yml"
for version in ("v9.8.7", "latest"):
    values = {
        "env.APP_NAME": "tokenlive-admin", "matrix.goos": "linux",
        "matrix.goarch": "amd64", "matrix.ext": "",
        "steps.version.outputs.VERSION": version,
    }
    with tempfile.TemporaryDirectory(prefix="admin-release-version-test-") as temporary:
        root = pathlib.Path(temporary)
        base_env = fixture(root)
        for name in ("Build frontend", "Build binary"):
            declared, command = step(release, name)
            env = {**base_env, **{key: resolve(value, values) for key, value in declared.items()}}
            result = run(resolve(command, values), root, env)
            if version == "latest":
                assert result.returncode != 0, f"release {name} accepted latest"
            else:
                assert result.returncode == 0, result.stderr
        if version != "latest":
            assert_metadata(root, version, "release")
        else:
            assert not entries(root), entries(root)

deploy = repo / ".github/workflows/deploy.yml"
values = {"github.sha": "0123456789abcdef"}
with tempfile.TemporaryDirectory(prefix="admin-deploy-version-test-") as temporary:
    root = pathlib.Path(temporary)
    env = fixture(root)
    # Read the deploy job's declared environment, not secret-bearing SSH steps.
    lines = deploy.read_text().splitlines()
    in_env = False
    for line in lines:
        if line == "    env:":
            in_env = True
            continue
        if in_env and line.startswith("      ") and ":" in line:
            key, value = line.strip().split(":", 1)
            env[key] = resolve(value.strip(), values)
        elif in_env:
            break
    for name in ("Build Frontend", "Compile Go Binary"):
        declared, command = step(deploy, name)
        result = run(resolve(command, values), root, {**env, **declared})
        assert result.returncode == 0, result.stderr
    assert_metadata(root, "git-0123456789abcdef", "dev")
PY

echo "version build tests passed"
