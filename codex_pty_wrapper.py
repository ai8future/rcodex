#!/usr/bin/env python3
"""
PTY wrapper for Codex resume command.
Handles terminal emulation requirements for non-interactive environments.
"""

import pty
import os
import sys
import subprocess
import select
import time
import fcntl
import struct
import termios
import re
import signal

def run_codex_resume(session_id, args, prompt, timeout=600):
    """Run codex resume with proper PTY support."""
    master, slave = pty.openpty()

    # Set terminal size
    s = struct.pack('HHHH', 50, 200, 0, 0)
    fcntl.ioctl(master, termios.TIOCSWINSZ, s)

    cmd = ['codex', 'resume', session_id] + args + [prompt]

    p = subprocess.Popen(
        cmd,
        stdin=slave,
        stdout=slave,
        stderr=slave,
        close_fds=True,
        env={**os.environ, 'TERM': 'xterm-256color'}
    )
    os.close(slave)

    output = b''
    start = time.time()
    last_output = start

    while time.time() - start < timeout:
        r, w, e = select.select([master], [], [], 0.5)
        if r:
            try:
                data = os.read(master, 8192)
                if data:
                    output += data
                    last_output = time.time()
                    # Respond to cursor position query (ESC[6n)
                    if b'\x1b[6n' in data:
                        os.write(master, b'\x1b[1;1R')
                    # Respond to device attributes query (ESC[c)
                    if b'\x1b[c' in data or b'\x1b[0c' in data:
                        os.write(master, b'\x1b[?1;2c')
            except OSError:
                break

        # Check if process has ended
        if p.poll() is not None:
            # Read any remaining output
            try:
                while True:
                    r, w, e = select.select([master], [], [], 0.1)
                    if not r:
                        break
                    data = os.read(master, 8192)
                    if not data:
                        break
                    output += data
            except:
                pass
            break

        # If no output for 30 seconds after start, process might be stuck
        if time.time() - last_output > 30 and time.time() - start > 30:
            break

    # Clean up
    try:
        os.close(master)
    except:
        pass

    if p.poll() is None:
        p.terminate()
        try:
            p.wait(timeout=5)
        except:
            p.kill()

    # Decode and clean output
    text = output.decode('utf-8', errors='ignore')

    # Remove ANSI escape sequences
    ansi_escape = re.compile(r'\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[\(\)][AB012]|\x1b[=>]')
    clean = ansi_escape.sub('', text)

    # Remove other control characters
    clean = re.sub(r'[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]', '', clean)

    # Clean up the output
    lines = clean.split('\n')
    result_lines = []
    in_result = False

    for line in lines:
        line = line.strip()
        if not line:
            continue
        # Skip UI elements
        if line.startswith('╭') or line.startswith('╰') or line.startswith('│'):
            continue
        if 'Update available' in line or 'Run npm install' in line:
            continue
        if line.startswith('Tip:') or line.startswith('›'):
            continue
        if line.startswith('model:') or line.startswith('directory:'):
            continue
        if 'OpenAI Codex' in line:
            continue
        # This is actual content
        if line:
            result_lines.append(line)

    return '\n'.join(result_lines), p.returncode

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print("Usage: codex_pty_wrapper.py <session_id> <prompt> [args...]", file=sys.stderr)
        sys.exit(1)

    session_id = sys.argv[1]
    prompt = sys.argv[2]
    extra_args = sys.argv[3:] if len(sys.argv) > 3 else []

    output, returncode = run_codex_resume(session_id, extra_args, prompt)
    print(output)
    sys.exit(returncode or 0)
