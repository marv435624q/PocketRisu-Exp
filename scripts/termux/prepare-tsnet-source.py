#!/usr/bin/env python3
"""Restore only the pinned candidate's patched Tailscale source; never overwrite edits."""
from pathlib import Path, PurePosixPath
import hashlib
import json
import tarfile

ROOT = Path(__file__).resolve().parent.parent.parent
source = ROOT / 'native/tsnet'
target = source / 'third_party/tailscale'
for parent in (source, source / 'third_party', target):
    assert not parent.is_symlink(), 'Source directory links are not supported'
record = json.loads((source / 'SOURCE.json').read_text(encoding='utf-8'))
archive_path = PurePosixPath(record['sourceArchive']['path'])
assert not archive_path.is_absolute() and '..' not in archive_path.parts
archive = ROOT / str(archive_path)
assert hashlib.sha256(archive.read_bytes()).hexdigest() == record['sourceArchive']['sha256'], 'Source archive hash mismatch'
prefix = record['sourceArchive']['prefix'] + '/native/tsnet/third_party/tailscale/'
with tarfile.open(archive, 'r:gz') as tar:
    selected = [member for member in tar.getmembers() if member.name.startswith(prefix)]
    assert selected, 'Patched Tailscale source missing from archive'
    expected = {}
    for member in selected:
        relative = PurePosixPath(member.name[len(prefix):])
        assert member.isfile() and not relative.is_absolute() and '..' not in relative.parts
        path = target / str(relative)
        assert all(not parent.is_symlink() for parent in (path, *path.parents) if parent.is_relative_to(target)), 'Source directory links are not supported'
        assert path.resolve().is_relative_to(target.resolve())
        expected[path] = tar.extractfile(member).read()
    existing = target
    if existing.exists():
        actual = {path for path in existing.rglob('*') if path.is_file()}
        assert actual <= set(expected), 'Extra source files found; use a new isolated worktree rather than deleting them'
    for path, raw in expected.items():
        if path.exists():
            assert path.read_bytes() == raw, f'Source edit retained; refusing to overwrite: {path.relative_to(ROOT)}'
        else:
            path.parent.mkdir(parents=True, exist_ok=True)
            with path.open('xb') as stream:
                stream.write(raw)
assert (source / 'third_party/tailscale/VERSION.txt').read_text().strip() == record['tailscale']
assert 'tailscale.com v' + record['tailscale'] in (source / 'go.mod').read_text()
print('Pinned patched source is present; no existing edits were overwritten')
