#!/usr/bin/env python3
"""
Run Codex commands in iTerm2 with proper terminal support.
Uses iTerm2's Python API to create a session, run the command, and capture output.
"""

import iterm2
import asyncio
import sys
import os
import tempfile
import time

async def main():
    if len(sys.argv) < 2:
        print("Usage: iterm_runner.py <command>", file=sys.stderr)
        sys.exit(1)

    command = sys.argv[1]
    output_file = sys.argv[2] if len(sys.argv) > 2 else None

    async with iterm2.Connection() as connection:
        app = await iterm2.App.async_get(running_or_new(connection))

        # Create a new window with a single tab
        window = await iterm2.Window.async_create(connection)
        if window is None:
            print("Failed to create iTerm2 window", file=sys.stderr)
            sys.exit(1)

        session = window.current_tab.current_session

        # Set up a marker for output capture
        marker_start = f"__RCODEGEN_START_{os.getpid()}__"
        marker_end = f"__RCODEGEN_END_{os.getpid()}__"

        # Send the command with markers
        full_cmd = f'echo "{marker_start}"; {command}; echo "{marker_end}"'
        await session.async_send_text(full_cmd + "\n")

        # Wait for the end marker
        output = ""
        max_wait = 600  # 10 minutes max
        start_time = time.time()

        while time.time() - start_time < max_wait:
            # Get the session's screen contents
            screen = await session.async_get_screen_contents()
            lines = []
            for i in range(screen.number_of_lines):
                line = screen.line(i)
                lines.append(line.string)

            full_output = "\n".join(lines)

            if marker_end in full_output:
                # Extract content between markers
                start_idx = full_output.find(marker_start)
                end_idx = full_output.find(marker_end)
                if start_idx != -1 and end_idx != -1:
                    output = full_output[start_idx + len(marker_start):end_idx].strip()
                break

            await asyncio.sleep(0.5)

        # Close the window
        await window.async_close(force=True)

        # Output the result
        if output_file:
            with open(output_file, 'w') as f:
                f.write(output)
        else:
            print(output)

def running_or_new(connection):
    return connection

if __name__ == "__main__":
    asyncio.run(main())
