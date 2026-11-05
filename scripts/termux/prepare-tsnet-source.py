#!/usr/bin/env python3
"""Restore pinned cert1 source, then apply exact cert2 diagnostic-only ACME markers."""
from pathlib import Path, PurePosixPath
import hashlib
import json
import tarfile

ROOT = Path(__file__).resolve().parent.parent.parent
source = ROOT / 'native/tsnet'
target = source / 'third_party/tailscale'


def replace_once(text, old, new, label):
    count = text.count(old)
    assert count == 1, f'{label}: expected one source match, got {count}'
    return text.replace(old, new, 1)


def add_cert2_acme_markers(raw):
    text = raw.decode('utf-8')
    text = replace_once(text,
        '\ta, err := e.ensureAccount(ctx, ac, logf, traceACME)\n\tif err != nil {\n\t\treturn nil, err\n\t}',
        '\tlogf("pocketrisu-cert-stage: account start")\n'
        '\ta, err := e.ensureAccount(ctx, ac, logf, traceACME)\n'
        '\tif err != nil {\n\t\tlogf("pocketrisu-cert-stage: account error")\n\t\treturn nil, err\n\t}\n'
        '\tlogf("pocketrisu-cert-stage: account done")',
        'account stage')
    text = replace_once(text,
        '\torder, err := ac.AuthorizeOrder(ctx, authzIDs, args.opts...)\n\tif err != nil {\n\t\treturn nil, err\n\t}',
        '\targs.logf("pocketrisu-cert-stage: order start")\n'
        '\torder, err := ac.AuthorizeOrder(ctx, authzIDs, args.opts...)\n'
        '\tif err != nil {\n\t\targs.logf("pocketrisu-cert-stage: order error")\n\t\treturn nil, err\n\t}\n'
        '\targs.logf("pocketrisu-cert-stage: order done")',
        'order stage')
    text = replace_once(text,
        '\t\tcase acmeChallengeDNS01:\n\t\t\tif err := fulfillACMEDNS01Challenge(ctx, b, ac, az, args.logf, args.traceACME); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}',
        '\t\tcase acmeChallengeDNS01:\n'
        '\t\t\targs.logf("pocketrisu-cert-stage: dns start")\n'
        '\t\t\tif err := fulfillACMEDNS01Challenge(ctx, b, ac, az, args.logf, args.traceACME); err != nil {\n'
        '\t\t\t\targs.logf("pocketrisu-cert-stage: dns error")\n\t\t\t\treturn nil, err\n\t\t\t}\n'
        '\t\t\targs.logf("pocketrisu-cert-stage: dns done")',
        'dns stage')
    text = replace_once(text,
        '\torderURI := order.URI\n\torder, err = ac.WaitOrder(ctx, orderURI)\n\tif err != nil {\n'
        '\t\tif ctx.Err() != nil {\n\t\t\treturn nil, ctx.Err()\n\t\t}\n'
        '\t\tif oe, ok := err.(*xacme.OrderError); ok {\n'
        '\t\t\targs.logf("acme: WaitOrder: OrderError status %q", oe.Status)\n'
        '\t\t} else {\n\t\t\targs.logf("acme: WaitOrder error: %v", err)\n\t\t}\n'
        '\t\treturn nil, err\n\t}\n\targs.traceACME(order)',
        '\torderURI := order.URI\n'
        '\targs.logf("pocketrisu-cert-stage: issuance-wait start")\n'
        '\torder, err = ac.WaitOrder(ctx, orderURI)\n\tif err != nil {\n'
        '\t\targs.logf("pocketrisu-cert-stage: issuance-wait error")\n'
        '\t\tif ctx.Err() != nil {\n\t\t\treturn nil, ctx.Err()\n\t\t}\n'
        '\t\tif oe, ok := err.(*xacme.OrderError); ok {\n'
        '\t\t\targs.logf("acme: WaitOrder: OrderError status %q", oe.Status)\n'
        '\t\t} else {\n\t\t\targs.logf("acme: WaitOrder error: %v", err)\n\t\t}\n'
        '\t\treturn nil, err\n\t}\n'
        '\targs.logf("pocketrisu-cert-stage: issuance-wait done")\n\targs.traceACME(order)',
        'issuance wait stage')
    text = replace_once(text,
        '\targs.logf("requesting cert...")\n\targs.traceACME(csr)\n'
        '\tder, _, err := ac.CreateOrderCert(ctx, order.FinalizeURL, csr, true)\n'
        '\tif err != nil {\n\t\treturn nil, fmt.Errorf("CreateOrder: %v", err)\n\t}\n'
        '\targs.logf("got cert")',
        '\targs.logf("pocketrisu-cert-stage: certificate start")\n'
        '\targs.logf("requesting cert...")\n\targs.traceACME(csr)\n'
        '\tder, _, err := ac.CreateOrderCert(ctx, order.FinalizeURL, csr, true)\n'
        '\tif err != nil {\n\t\targs.logf("pocketrisu-cert-stage: certificate error")\n'
        '\t\treturn nil, fmt.Errorf("CreateOrder: %v", err)\n\t}\n'
        '\targs.logf("pocketrisu-cert-stage: certificate done")\n\targs.logf("got cert")',
        'certificate stage')
    return text.encode('utf-8')


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
    baseline = {}
    for member in selected:
        relative = PurePosixPath(member.name[len(prefix):])
        assert member.isfile() and not relative.is_absolute() and '..' not in relative.parts
        path = target / str(relative)
        assert all(not parent.is_symlink() for parent in (path, *path.parents) if parent.is_relative_to(target)), 'Source directory links are not supported'
        assert path.resolve().is_relative_to(target.resolve())
        baseline[path] = tar.extractfile(member).read()

expected = dict(baseline)
cert_file = target / 'feature/acme/cert.go'
assert cert_file in expected, 'Pinned ACME source missing'
expected[cert_file] = add_cert2_acme_markers(expected[cert_file])

if target.exists():
    actual_files = {path for path in target.rglob('*') if path.is_file()}
    assert actual_files <= set(expected), 'Extra source files found; use a new isolated worktree rather than deleting them'

for path, raw in expected.items():
    if path.exists():
        actual = path.read_bytes()
        if actual == raw:
            continue
        if actual == baseline[path] and raw != baseline[path]:
            path.write_bytes(raw)
            continue
        raise AssertionError(f'Source edit retained; refusing to overwrite: {path.relative_to(ROOT)}')
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('xb') as stream:
        stream.write(raw)

assert (target / 'VERSION.txt').read_text().strip() == record['tailscale']
assert 'tailscale.com v' + record['tailscale'] in (source / 'go.mod').read_text()
assert b'pocketrisu-cert-stage: certificate done' in cert_file.read_bytes()
print('Pinned cert1 source restored and exact cert2 diagnostic markers are present; unknown edits were not overwritten')
