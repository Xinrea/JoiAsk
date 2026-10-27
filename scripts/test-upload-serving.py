#!/usr/bin/env python3
"""HTTP regression test. Requires caddy, node, and `cd frontend && npm run build`."""

import base64
import contextlib
import pathlib
import socket
import subprocess
import tempfile
import time
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[1]
PUBLIC = ROOT / "frontend/public/upload-img"
CSP = "sandbox; default-src 'none'; img-src data:; style-src 'unsafe-inline'"
PNG = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aCHsAAAAASUVORK5CYII="
)
SVG = b'<svg xmlns="http://www.w3.org/2000/svg"><style>rect{fill:red}</style><rect width="10" height="10"/><script>alert(document.domain)</script></svg>'
HTML = b"<!doctype html><script>alert(document.domain)</script>"
CASES = [
    ("image.png", PNG, "image/png", "inline"),
    ("image.JPG", PNG, "image/jpeg", "inline"),
    ("image.jpeg", PNG, "image/jpeg", "inline"),
    ("image.gif", PNG, "image/gif", "inline"),
    ("image.webp", PNG, "image/webp", "inline"),
    ("site-asset.svg", SVG, "image/svg+xml", "inline"),
    ("legacy.SVG", SVG, "image/svg+xml", "inline"),
    ("legacy.html", HTML, "application/octet-stream", "attachment"),
    ("image.png.html", HTML, "application/octet-stream", "attachment"),
    ("imageXpng", HTML, "application/octet-stream", "attachment"),
    ("legacy.svgz", HTML, "application/octet-stream", "attachment"),
    # A misleading image suffix must never opt out of the sandbox/nosniff.
    ("misleading.png", HTML, "image/png", "inline"),
]


def available_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


@contextlib.contextmanager
def server(command, port, log):
    with log.open("w+") as output:
        process = subprocess.Popen(command, cwd=ROOT, stdout=output, stderr=output)
        try:
            for _ in range(200):
                if process.poll() is not None:
                    output.seek(0)
                    raise RuntimeError(output.read())
                try:
                    with socket.create_connection(("127.0.0.1", port), timeout=0.1):
                        break
                except OSError:
                    time.sleep(0.1)
            else:
                raise RuntimeError(f"Server did not start: {command}")
            yield
        finally:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()


def check(port, prefix):
    for name, content, content_type, disposition in CASES:
        for method in ("GET", "HEAD"):
            url = f"http://127.0.0.1:{port}/upload-img/{prefix}/{name}?cache-test=1"
            request = urllib.request.Request(url, method=method)
            with urllib.request.urlopen(request, timeout=5) as response:
                assert response.status == 200, (url, response.status)
                headers = response.headers
                assert headers["Content-Security-Policy"] == CSP, (url, headers)
                assert headers["X-Content-Type-Options"] == "nosniff", (url, headers)
                assert headers.get_content_type() == content_type, (url, headers)
                assert headers["Content-Disposition"] == disposition, (url, headers)
                if method == "GET":
                    assert response.read() == content, url
    # Percent-encoded extensions must not bypass the policy.
    url = f"http://127.0.0.1:{port}/upload-img/{prefix}/legacy.%68tml"
    with urllib.request.urlopen(url, timeout=5) as response:
        assert response.headers.get_content_type() == "application/octet-stream"
        assert response.headers["Content-Disposition"] == "attachment"
        assert response.headers["Content-Security-Policy"] == CSP


def main():
    if not (ROOT / "frontend/.next/BUILD_ID").exists():
        raise SystemExit("First run: cd frontend && npm ci && npm run build")
    PUBLIC.mkdir(parents=True, exist_ok=True)
    # Both fixture and server files stay in this worktree; never touch real uploads.
    with tempfile.TemporaryDirectory(prefix="serving-test-", dir=PUBLIC) as fixtures, \
            tempfile.TemporaryDirectory(prefix=".serving-test-", dir=ROOT) as temporary:
        fixture_dir = pathlib.Path(fixtures)
        temp_dir = pathlib.Path(temporary)
        for name, content, _, _ in CASES:
            (fixture_dir / name).write_bytes(content)
        for filename, upload_root in (
            ("Caddyfile", "/work/frontend/public/upload-img"),
            ("Caddyfile.dev", "/srv/uploads"),
        ):
            port = available_port()
            config = (ROOT / filename).read_text()
            config = config.replace(":80 {", f"http://127.0.0.1:{port} {{")
            config = config.replace(upload_root, f'"{PUBLIC}"')
            config_path = temp_dir / filename
            config_path.write_text(config)
            with server(["caddy", "run", "--adapter", "caddyfile", "--config", str(config_path)],
                        port, temp_dir / f"{filename}.log"):
                check(port, fixture_dir.name)
            print(f"PASS {filename}: upload GET/HEAD MIME, disposition, CSP and nosniff")
        port = available_port()
        with server(["node", "frontend/node_modules/next/dist/bin/next", "start", "frontend",
                     "--hostname", "127.0.0.1", "--port", str(port)],
                    port, temp_dir / "next.log"):
            check(port, fixture_dir.name)
        print("PASS direct Next: upload GET/HEAD MIME, disposition, CSP and nosniff")


if __name__ == "__main__":
    main()
