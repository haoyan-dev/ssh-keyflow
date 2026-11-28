"""Throwaway spike for ticket 01: probe `ssh-keygen -Y` roster signing.

Not product code. Usage: python -I openssh_sign_spike.py [WORK_DIR]
Prints one JSON report. Uses only temporary keys in WORK_DIR (default: a new
temp dir) and a private ssh-agent where one can be started; never touches the
user's keys or agent.
"""
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
from pathlib import Path

NS = "ssh-keyflow-roster"
PASS = "spike-pass-123"
WIN = os.name == "nt"
work = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(tempfile.mkdtemp(prefix="kf-spike-"))
work.mkdir(parents=True, exist_ok=True)
report = {"platform": sys.platform, "cases": {}}


def run(args, data=None, env=None, detach=False, timeout=20):
    kw = {"capture_output": True, "timeout": timeout}
    if detach:
        kw["stdin"] = subprocess.DEVNULL
        if WIN:
            kw["creationflags"] = subprocess.DETACHED_PROCESS | subprocess.CREATE_NEW_PROCESS_GROUP
        else:
            kw["start_new_session"] = True  # no controlling TTY
    elif data is not None:
        kw["input"] = data
    else:
        kw["stdin"] = subprocess.DEVNULL
    full_env = dict(os.environ)
    for k in ("SSH_AUTH_SOCK", "SSH_ASKPASS", "SSH_ASKPASS_REQUIRE", "DISPLAY"):
        full_env.pop(k, None)
    full_env.update(env or {})
    try:
        p = subprocess.run(args, env=full_env, **kw)
        return {"exit": p.returncode, "stdout": p.stdout.decode(errors="replace").strip(),
                "stderr": p.stderr.decode(errors="replace").strip()}
    except subprocess.TimeoutExpired:
        return {"exit": "timeout"}


def case(name, result, expect_ok):
    result["expected"] = "success" if expect_ok else "failure"
    result["as_expected"] = (result.get("exit") == 0) == expect_ok
    report["cases"][name] = result


version = run(["ssh", "-V"])
report["openssh"] = version["stderr"] or version["stdout"]

key, plain, other = work / "root", work / "plain", work / "other"
for k in (key, plain, other):
    for f in (k, k.with_suffix(".pub")):
        f.unlink(missing_ok=True)
run(["ssh-keygen", "-q", "-t", "ed25519", "-N", PASS, "-C", "fleet-root", "-f", str(key)])
run(["ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "plain", "-f", str(plain)])
run(["ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "other", "-f", str(other)])

roster = work / "roster.json"
roster.write_bytes(b'{"version":1,"nodes":[]}\n')
allowed = work / "allowed_signers"
pub = lambda k: k.with_suffix(".pub").read_text().split()[:2]
allowed.write_text(f'fleet-root namespaces="{NS}" {" ".join(pub(key))}\n'
                   f'plain namespaces="{NS}" {" ".join(pub(plain))}\n')


def sign(k, out_name, **kw):
    sig = roster.with_name(roster.name + ".sig")
    sig.unlink(missing_ok=True)
    r = run(["ssh-keygen", "-Y", "sign", "-f", str(k), "-n", NS, str(roster)], **kw)
    if sig.exists():
        shutil.copy(sig, work / out_name)
    r["sig_written"] = sig.exists()
    return r


def verify(sig, principal="plain", ns=NS, data=None, signers=allowed):
    return run(["ssh-keygen", "-Y", "verify", "-f", str(signers), "-I", principal,
                "-n", ns, "-s", str(work / sig)], data=data if data is not None else roster.read_bytes())


# Baseline: unencrypted key, then verification variants.
case("sign_unencrypted", sign(plain, "plain.sig"), True)
case("verify_good", verify("plain.sig"), True)
case("verify_tampered", verify("plain.sig", data=roster.read_bytes() + b" "), False)
case("verify_crlf_changed", verify("plain.sig", data=roster.read_bytes().replace(b"\n", b"\r\n")), False)
case("verify_wrong_namespace", verify("plain.sig", ns="other-namespace"), False)
case("verify_unknown_principal", verify("plain.sig", principal="nobody"), False)
case("verify_wrong_principal_for_key", verify("plain.sig", principal="fleet-root"), False)
sign(other, "other.sig")
case("verify_unlisted_key", verify("other.sig", principal="plain"), False)
restricted = work / "allowed_other_ns"
restricted.write_text(f'plain namespaces="file" {" ".join(pub(plain))}\n')
case("verify_signer_namespace_restriction", verify("plain.sig", signers=restricted), False)
case("find_principals", run(["ssh-keygen", "-Y", "find-principals", "-f", str(allowed),
                             "-s", str(work / "plain.sig")]), True)
case("check_novalidate", run(["ssh-keygen", "-Y", "check-novalidate", "-n", NS, "-s",
                              str(work / "plain.sig")], data=roster.read_bytes()), True)
case("sig_format", {"exit": 0, "first_line": (work / "plain.sig").read_text().splitlines()[0]}, True)

# Passphrase-protected key without a terminal.
case("sign_passphrase_no_tty", sign(key, "nt.sig", detach=True), False)
if WIN:
    helper = work / "askpass.cmd"
    helper.write_text(f"@echo {PASS}\r\n")
else:
    helper = work / "askpass.sh"
    helper.write_text(f"#!/bin/sh\necho '{PASS}'\n")
    helper.chmod(0o700)
askenv = {"SSH_ASKPASS": str(helper), "SSH_ASKPASS_REQUIRE": "force", "DISPLAY": ":0"}
r = sign(key, "askpass.sig", env=askenv, detach=True)
case("sign_passphrase_askpass_force", r, True)
if r["sig_written"]:
    case("verify_askpass_sig", verify("askpass.sig", principal="fleet-root"), True)

# Signing through a private ssh-agent with only the public key file given.
agent_env, agent_proc = None, None
if WIN:
    pipe = rf"\\.\pipe\kf-spike-{os.getpid()}"
    try:
        agent_proc = subprocess.Popen(["ssh-agent", "-d", "-a", pipe],
                                      stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        time.sleep(1.5)
        if agent_proc.poll() is None:
            agent_env = {"SSH_AUTH_SOCK": pipe}
        else:
            report["agent_start"] = agent_proc.stderr.read().decode(errors="replace").strip()
    except OSError as e:
        report["agent_start"] = str(e)
else:
    sock = work / "agent.sock"
    sock.unlink(missing_ok=True)
    agent_proc = subprocess.Popen(["ssh-agent", "-D", "-a", str(sock)],
                                  stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    time.sleep(1)
    agent_env = {"SSH_AUTH_SOCK": str(sock)}
if agent_env:
    add = run(["ssh-add", str(key)], env={**agent_env, **askenv}, detach=True)
    case("agent_add_with_askpass", add, True)
    r = run(["ssh-keygen", "-Y", "sign", "-f", str(key.with_suffix(".pub")), "-n", NS, str(roster)],
            env=agent_env, detach=True)
    sig = roster.with_name(roster.name + ".sig")
    r["sig_written"] = sig.exists()
    if sig.exists():
        shutil.copy(sig, work / "agent.sig")
    case("sign_via_agent_pubkey_only", r, True)
    if r["sig_written"]:
        case("verify_agent_sig", verify("agent.sig", principal="fleet-root"), True)
    run(["ssh-add", "-D"], env=agent_env)
if agent_proc:
    agent_proc.terminate()
    agent_proc.wait(timeout=10)

report["all_as_expected"] = all(c["as_expected"] for c in report["cases"].values())
report["work_dir"] = str(work)
print(json.dumps(report, indent=2))
