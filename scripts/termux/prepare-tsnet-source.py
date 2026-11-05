#!/usr/bin/env python3
"""Restore pinned cert1 source, then apply exact cert3 diagnostic-only ACME/control markers."""
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
        '\ta, err := e.ensureAccount(ctx, ac, logf, traceACME)\n'
        '\tif err != nil {\n\t\treturn nil, err\n\t}\n'
        '\tif a.Status != xacme.StatusValid {\n'
        '\t\treturn nil, fmt.Errorf("unexpected ACME account status %q", a.Status)\n\t}',
        '\tdiagnostic := minValidity == time.Nanosecond\n'
        '\tcert2DiagnosticLog(logf, diagnostic, "account start")\n'
        '\ta, err := e.ensureAccount(ctx, ac, logf, traceACME)\n'
        '\tif err != nil {\n'
        '\t\tcert2DiagnosticLog(logf, diagnostic, "account error")\n'
        '\t\treturn nil, err\n\t}\n'
        '\tif a.Status != xacme.StatusValid {\n'
        '\t\tcert2DiagnosticLog(logf, diagnostic, "account error")\n'
        '\t\treturn nil, fmt.Errorf("unexpected ACME account status %q", a.Status)\n\t}\n'
        '\tcert2DiagnosticLog(logf, diagnostic, "account done")',
        'account stage')

    text = replace_once(text,
        '\tissueArgs := acmeCertIssueArgs{\n'
        '\t\tcs:        cs,\n'
        '\t\tlogf:      logf,\n'
        '\t\ttraceACME: traceACME,\n'
        '\t\tdomain:    domain,\n'
        '\t\topts:      opts,\n'
        '\t}',
        '\tissueArgs := acmeCertIssueArgs{\n'
        '\t\tcs:         cs,\n'
        '\t\tlogf:       logf,\n'
        '\t\ttraceACME:  traceACME,\n'
        '\t\tdomain:     domain,\n'
        '\t\topts:       opts,\n'
        '\t\tdiagnostic: diagnostic,\n'
        '\t}',
        'diagnostic issue args')

    text = replace_once(text,
        '\tchallengeType acmeChallengeType   // challenge type to fulfill\n'
        '}\n\n'
        'func (args acmeCertIssueArgs) baseDomain() string { return strings.TrimPrefix(args.domain, "*.") }\n'
        'func (args acmeCertIssueArgs) isWildcard() bool   { return isWildcardDomain(args.domain) }',
        '\tchallengeType acmeChallengeType   // challenge type to fulfill\n'
        '\tdiagnostic    bool                // true only for PocketRisu explicit cert2 verification\n'
        '}\n\n'
        'func (args acmeCertIssueArgs) baseDomain() string { return strings.TrimPrefix(args.domain, "*.") }\n'
        'func (args acmeCertIssueArgs) isWildcard() bool   { return isWildcardDomain(args.domain) }\n\n'
        'func cert2DiagnosticLog(logf logger.Logf, enabled bool, event string) {\n'
        '\tif enabled {\n'
        '\t\tlogf("pocketrisu-cert-stage: " + event)\n'
        '\t}\n'
        '}',
        'diagnostic helper')

    text = replace_once(text,
        '\torder, err := ac.AuthorizeOrder(ctx, authzIDs, args.opts...)\n'
        '\tif err != nil {\n\t\treturn nil, err\n\t}',
        '\tcert2DiagnosticLog(args.logf, args.diagnostic, "order start")\n'
        '\torder, err := ac.AuthorizeOrder(ctx, authzIDs, args.opts...)\n'
        '\tif err != nil {\n'
        '\t\tcert2DiagnosticLog(args.logf, args.diagnostic, "order error")\n'
        '\t\treturn nil, err\n\t}\n'
        '\tcert2DiagnosticLog(args.logf, args.diagnostic, "order done")',
        'order stage')

    text = replace_once(text,
        '\t\tcase acmeChallengeDNS01:\n'
        '\t\t\tif err := fulfillACMEDNS01Challenge(ctx, b, ac, az, args.logf, args.traceACME); err != nil {\n'
        '\t\t\t\treturn nil, err\n'
        '\t\t\t}',
        '\t\tcase acmeChallengeDNS01:\n'
        '\t\t\tcert2DiagnosticLog(args.logf, args.diagnostic, "dns start")\n'
        '\t\t\tif err := fulfillACMEDNS01Challenge(ctx, b, ac, az, args.logf, args.traceACME); err != nil {\n'
        '\t\t\t\tcert2DiagnosticLog(args.logf, args.diagnostic, "dns error")\n'
        '\t\t\t\treturn nil, err\n'
        '\t\t\t}\n'
        '\t\t\tcert2DiagnosticLog(args.logf, args.diagnostic, "dns done")',
        'dns stage')

    text = replace_once(text,
        '\torderURI := order.URI\n'
        '\torder, err = ac.WaitOrder(ctx, orderURI)\n'
        '\tif err != nil {\n'
        '\t\tif ctx.Err() != nil {\n\t\t\treturn nil, ctx.Err()\n\t\t}\n'
        '\t\tif oe, ok := err.(*xacme.OrderError); ok {\n'
        '\t\t\targs.logf("acme: WaitOrder: OrderError status %q", oe.Status)\n'
        '\t\t} else {\n'
        '\t\t\targs.logf("acme: WaitOrder error: %v", err)\n'
        '\t\t}\n'
        '\t\treturn nil, err\n'
        '\t}\n'
        '\targs.traceACME(order)',
        '\torderURI := order.URI\n'
        '\tcert2DiagnosticLog(args.logf, args.diagnostic, "issuance-wait start")\n'
        '\torder, err = ac.WaitOrder(ctx, orderURI)\n'
        '\tif err != nil {\n'
        '\t\tcert2DiagnosticLog(args.logf, args.diagnostic, "issuance-wait error")\n'
        '\t\tif ctx.Err() != nil {\n\t\t\treturn nil, ctx.Err()\n\t\t}\n'
        '\t\tif oe, ok := err.(*xacme.OrderError); ok {\n'
        '\t\t\targs.logf("acme: WaitOrder: OrderError status %q", oe.Status)\n'
        '\t\t} else {\n'
        '\t\t\targs.logf("acme: WaitOrder error: %v", err)\n'
        '\t\t}\n'
        '\t\treturn nil, err\n'
        '\t}\n'
        '\tcert2DiagnosticLog(args.logf, args.diagnostic, "issuance-wait done")\n'
        '\targs.traceACME(order)',
        'issuance wait stage')

    text = replace_once(text,
        '\targs.logf("requesting cert...")\n'
        '\targs.traceACME(csr)\n'
        '\tder, _, err := ac.CreateOrderCert(ctx, order.FinalizeURL, csr, true)\n'
        '\tif err != nil {\n\t\treturn nil, fmt.Errorf("CreateOrder: %v", err)\n\t}\n'
        '\targs.logf("got cert")',
        '\tcert2DiagnosticLog(args.logf, args.diagnostic, "certificate start")\n'
        '\targs.logf("requesting cert...")\n'
        '\targs.traceACME(csr)\n'
        '\tder, _, err := ac.CreateOrderCert(ctx, order.FinalizeURL, csr, true)\n'
        '\tif err != nil {\n'
        '\t\tcert2DiagnosticLog(args.logf, args.diagnostic, "certificate error")\n'
        '\t\treturn nil, fmt.Errorf("CreateOrder: %v", err)\n\t}\n'
        '\tcert2DiagnosticLog(args.logf, args.diagnostic, "certificate done")\n'
        '\targs.logf("got cert")',
        'certificate stage')

    return text.encode('utf-8')


def add_cert3_acme_substages(raw):
    text = raw.decode('utf-8')
    text = text.replace('cert2DiagnosticLog', 'cert3DiagnosticLog')
    text = text.replace('true only for PocketRisu explicit cert2 verification',
                        'true only for PocketRisu explicit cert3 verification')

    text = replace_once(text,
        '\t\t\tif err := fulfillACMEDNS01Challenge(ctx, b, ac, az, args.logf, args.traceACME); err != nil {\n',
        '\t\t\tif err := fulfillACMEDNS01Challenge(ctx, b, ac, az, args.logf, args.traceACME, args.diagnostic); err != nil {\n',
        'cert3 dns diagnostic argument')

    text = replace_once(text,
        'func fulfillACMEDNS01Challenge(ctx context.Context, b *ipnlocal.LocalBackend, ac *xacme.Client, az *xacme.Authorization, logf logger.Logf, traceACME func(any)) error {\n',
        'func fulfillACMEDNS01Challenge(ctx context.Context, b *ipnlocal.LocalBackend, ac *xacme.Client, az *xacme.Authorization, logf logger.Logf, traceACME func(any), diagnostic bool) error {\n',
        'cert3 dns helper signature')

    text = replace_once(text,
        '\t\tvar resolver net.Resolver\n'
        '\t\tlookupCtx, lookupCancel := context.WithTimeout(ctx, 500*time.Millisecond)\n'
        '\t\ttxts, _ := resolver.LookupTXT(lookupCtx, key)\n'
        '\t\tlookupCancel()\n',
        '\t\tvar resolver net.Resolver\n'
        '\t\tcert3DiagnosticLog(logf, diagnostic, "dns-lookup start")\n'
        '\t\tlookupCtx, lookupCancel := context.WithTimeout(ctx, 500*time.Millisecond)\n'
        '\t\ttxts, _ := resolver.LookupTXT(lookupCtx, key)\n'
        '\t\tlookupCancel()\n'
        '\t\tcert3DiagnosticLog(logf, diagnostic, "dns-lookup done")\n',
        'cert3 dns lookup')

    text = replace_once(text,
        '\t\t\tlogf("starting SetDNS call for %s...", key)\n'
        '\t\t\terr = b.SetDNS(ctx, key, rec)\n'
        '\t\t\tif err != nil {\n'
        '\t\t\t\treturn fmt.Errorf("SetDNS %q => %q: %w", key, rec, err)\n'
        '\t\t\t}\n'
        '\t\t\tlogf("did SetDNS for %s", key)\n',
        '\t\t\tlogf("starting SetDNS call for %s...", key)\n'
        '\t\t\tcert3DiagnosticLog(logf, diagnostic, "set-dns start")\n'
        '\t\t\terr = b.SetDNS(ctx, key, rec)\n'
        '\t\t\tif err != nil {\n'
        '\t\t\t\tcert3DiagnosticLog(logf, diagnostic, "set-dns error")\n'
        '\t\t\t\treturn fmt.Errorf("SetDNS %q => %q: %w", key, rec, err)\n'
        '\t\t\t}\n'
        '\t\t\tcert3DiagnosticLog(logf, diagnostic, "set-dns done")\n'
        '\t\t\tlogf("did SetDNS for %s", key)\n',
        'cert3 set-dns wrapper')

    text = replace_once(text,
        '\t\tchal, err := ac.Accept(ctx, ch)\n'
        '\t\tif err != nil {\n'
        '\t\t\treturn fmt.Errorf("Accept: %v", err)\n'
        '\t\t}\n'
        '\t\ttraceACME(chal)\n',
        '\t\tcert3DiagnosticLog(logf, diagnostic, "acme-accept start")\n'
        '\t\tchal, err := ac.Accept(ctx, ch)\n'
        '\t\tif err != nil {\n'
        '\t\t\tcert3DiagnosticLog(logf, diagnostic, "acme-accept error")\n'
        '\t\t\treturn fmt.Errorf("Accept: %v", err)\n'
        '\t\t}\n'
        '\t\tcert3DiagnosticLog(logf, diagnostic, "acme-accept done")\n'
        '\t\ttraceACME(chal)\n',
        'cert3 acme accept')
    return text.encode('utf-8')


def add_cert3_control_substages(raw):
    text = raw.decode('utf-8')
    text = replace_once(text,
        '// setDNSNoise sends the SetDNSRequest request to the control plane server over Noise,\n',
        'func cert3ControlDiagnosticLog(logf logger.Logf, event string) {\n'
        '\tif os.Getenv("POCKETRISU_CERT3_DIAGNOSTICS") == "1" {\n'
        '\t\tlogf("pocketrisu-cert-stage: " + event)\n'
        '\t}\n'
        '}\n\n'
        '// setDNSNoise sends the SetDNSRequest request to the control plane server over Noise,\n',
        'cert3 control helper')

    text = replace_once(text,
        'func (c *Direct) setDNSNoise(ctx context.Context, req *tailcfg.SetDNSRequest) error {\n'
        '\tif err := c.ensureServerNoiseKey(ctx); err != nil {\n'
        '\t\treturn err\n'
        '\t}\n'
        '\tnewReq := *req\n',
        'func (c *Direct) setDNSNoise(ctx context.Context, req *tailcfg.SetDNSRequest) error {\n'
        '\tcert3ControlDiagnosticLog(c.logf, "noise-key start")\n'
        '\tif err := c.ensureServerNoiseKey(ctx); err != nil {\n'
        '\t\tcert3ControlDiagnosticLog(c.logf, "noise-key error")\n'
        '\t\treturn err\n'
        '\t}\n'
        '\tcert3ControlDiagnosticLog(c.logf, "noise-key done")\n'
        '\tnewReq := *req\n',
        'cert3 noise key')

    text = replace_once(text,
        '\tnc, err := c.getNoiseClient()\n'
        '\tif err != nil {\n'
        '\t\treturn err\n'
        '\t}\n'
        '\tres, err := nc.Post(ctx, "/machine/set-dns", newReq.NodeKey, &newReq)\n'
        '\tif err != nil {\n'
        '\t\treturn err\n'
        '\t}\n',
        '\tcert3ControlDiagnosticLog(c.logf, "noise-client start")\n'
        '\tnc, err := c.getNoiseClient()\n'
        '\tif err != nil {\n'
        '\t\tcert3ControlDiagnosticLog(c.logf, "noise-client error")\n'
        '\t\treturn err\n'
        '\t}\n'
        '\tcert3ControlDiagnosticLog(c.logf, "noise-client done")\n'
        '\tcert3ControlDiagnosticLog(c.logf, "set-dns-post start")\n'
        '\tres, err := nc.Post(ctx, "/machine/set-dns", newReq.NodeKey, &newReq)\n'
        '\tif err != nil {\n'
        '\t\tcert3ControlDiagnosticLog(c.logf, "set-dns-post error")\n'
        '\t\treturn err\n'
        '\t}\n'
        '\tcert3ControlDiagnosticLog(c.logf, "set-dns-post done")\n',
        'cert3 noise client and post')

    text = replace_once(text,
        '\tdefer res.Body.Close()\n'
        '\tif res.StatusCode != 200 {\n'
        '\t\tmsg, _ := io.ReadAll(res.Body)\n'
        '\t\treturn fmt.Errorf("set-dns response: %v, %.200s", res.Status, strings.TrimSpace(string(msg)))\n'
        '\t}\n'
        '\tvar setDNSRes tailcfg.SetDNSResponse\n'
        '\tif err := json.NewDecoder(res.Body).Decode(&setDNSRes); err != nil {\n'
        '\t\tc.logf("error decoding SetDNSResponse: %v", err)\n'
        '\t\treturn fmt.Errorf("set-dns-response: %w", err)\n'
        '\t}\n\n'
        '\treturn nil\n',
        '\tdefer res.Body.Close()\n'
        '\tcert3ControlDiagnosticLog(c.logf, "set-dns-response start")\n'
        '\tif res.StatusCode != 200 {\n'
        '\t\tmsg, _ := io.ReadAll(res.Body)\n'
        '\t\tcert3ControlDiagnosticLog(c.logf, "set-dns-response error")\n'
        '\t\treturn fmt.Errorf("set-dns response: %v, %.200s", res.Status, strings.TrimSpace(string(msg)))\n'
        '\t}\n'
        '\tvar setDNSRes tailcfg.SetDNSResponse\n'
        '\tif err := json.NewDecoder(res.Body).Decode(&setDNSRes); err != nil {\n'
        '\t\tc.logf("error decoding SetDNSResponse: %v", err)\n'
        '\t\tcert3ControlDiagnosticLog(c.logf, "set-dns-response error")\n'
        '\t\treturn fmt.Errorf("set-dns-response: %w", err)\n'
        '\t}\n'
        '\tcert3ControlDiagnosticLog(c.logf, "set-dns-response done")\n\n'
        '\treturn nil\n',
        'cert3 set-dns response')
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
control_file = target / 'control/controlclient/direct.go'
assert cert_file in expected, 'Pinned ACME source missing'
assert control_file in expected, 'Pinned control client source missing'
expected[cert_file] = add_cert3_acme_substages(add_cert2_acme_markers(expected[cert_file]))
expected[control_file] = add_cert3_control_substages(expected[control_file])

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
cert_bytes = cert_file.read_bytes()
control_bytes = control_file.read_bytes()
assert b'pocketrisu-cert-stage: ' in cert_bytes
assert b'cert3DiagnosticLog(args.logf, args.diagnostic, "certificate done")' in cert_bytes
assert b'cert3DiagnosticLog(logf, diagnostic, "dns-lookup start")' in cert_bytes
assert b'cert3DiagnosticLog(logf, diagnostic, "set-dns start")' in cert_bytes
assert b'cert3DiagnosticLog(logf, diagnostic, "acme-accept start")' in cert_bytes
assert b'cert3ControlDiagnosticLog(c.logf, "noise-key start")' in control_bytes
assert b'cert3ControlDiagnosticLog(c.logf, "set-dns-post start")' in control_bytes
assert b'cert3ControlDiagnosticLog(c.logf, "set-dns-response start")' in control_bytes
print('Pinned cert1 source restored and exact cert3 diagnostic markers are present; unknown edits were not overwritten')
