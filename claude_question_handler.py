#!/usr/bin/env python3
"""
claude_question_handler.py - Detect and answer Claude Code multiple choice questions via iTerm2

Can connect to an existing Claude session to:
1. Detect if a multiple choice question is displayed
2. Extract the question and options
3. Send an answer (by option number or text)

Usage:
    # Detect question in current session
    python3 claude_question_handler.py detect

    # Answer with option number (1-based)
    python3 claude_question_handler.py answer 2

    # Answer with custom text (for "Type something" option)
    python3 claude_question_handler.py answer --text "my custom answer"

    # Get JSON output of detected question
    python3 claude_question_handler.py detect --json

Requirements:
    - iTerm2 with Python API enabled
    - iterm2 Python package: pip install iterm2
"""

import asyncio
import json
import re
import sys
import os
import argparse
from dataclasses import dataclass, asdict
from typing import Optional

# Check for iTerm2 environment
if not os.environ.get('ITERM_SESSION_ID'):
    print(json.dumps({
        "error": "not_iterm2",
        "message": "Not running in iTerm2."
    }))
    sys.exit(1)

try:
    import iterm2
except ImportError:
    print(json.dumps({
        "error": "no_iterm2_package",
        "message": "iterm2 Python package not installed. Run: pip install iterm2"
    }))
    sys.exit(1)


@dataclass
class QuestionOption:
    number: int
    label: str
    description: str
    is_selected: bool
    is_text_input: bool  # True for "Type something" option


@dataclass
class DetectedQuestion:
    header: str  # e.g., "Next task"
    question: str  # The actual question text
    options: list  # List of QuestionOption
    selected_index: int  # 0-based index of currently selected option
    has_text_option: bool  # True if there's a "Type something" option


def strip_ansi(text: str) -> str:
    """Remove ANSI escape codes from text."""
    ansi_escape = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
    return ansi_escape.sub('', text)


def parse_multiple_choice(screen_text: str) -> Optional[DetectedQuestion]:
    """Parse screen text to detect and extract multiple choice question.

    Format detected:
    ☐ Header text

    Question text here?

    ❯ 1. Option label
         Option description
      2. Another option
         Its description
      ...
      N. Type something.

    Enter to select · Tab/Arrow keys to navigate · Esc to cancel
    """
    lines = screen_text.split('\n')

    # Look for the checkbox header (☐)
    header = None
    header_idx = -1
    for i, line in enumerate(lines):
        # Check for ☐ or similar checkbox characters
        if '☐' in line or '☑' in line:
            # Extract header text after checkbox
            match = re.search(r'[☐☑]\s*(.+)', line)
            if match:
                header = match.group(1).strip()
                header_idx = i
                break

    if header_idx == -1:
        return None

    # Look for the question (non-empty line after header, before options)
    question = None
    question_idx = -1
    for i in range(header_idx + 1, min(header_idx + 10, len(lines))):
        line = lines[i].strip()
        # Skip empty lines
        if not line:
            continue
        # Stop if we hit an option (starts with ❯ or number)
        if re.match(r'^[❯>\s]*\d+\.', line):
            break
        # This might be the question
        if line and not line.startswith('Enter to select'):
            question = line
            question_idx = i
            break

    if not question:
        return None

    # Parse options - look for numbered items with ❯ or > selector
    options = []
    selected_index = 0
    current_option = None

    for i in range(question_idx + 1, len(lines)):
        line = lines[i]

        # Stop at navigation hints
        if 'Enter to select' in line or 'Tab/Arrow' in line:
            break

        # Check for option line: "❯ N. Label" or "  N. Label"
        # The selector can be ❯, >, or just spaces
        option_match = re.match(r'^(\s*[❯>]?\s*)(\d+)\.\s*(.+)$', line)
        if option_match:
            # Save previous option if exists
            if current_option:
                options.append(current_option)

            prefix = option_match.group(1)
            num = int(option_match.group(2))
            label = option_match.group(3).strip()
            is_selected = '❯' in prefix or '>' in prefix
            is_text_input = 'type something' in label.lower()

            if is_selected:
                selected_index = len(options)

            current_option = QuestionOption(
                number=num,
                label=label,
                description="",
                is_selected=is_selected,
                is_text_input=is_text_input
            )
        elif current_option and line.strip():
            # This is a description line for the current option
            # Usually indented under the label
            desc = line.strip()
            if current_option.description:
                current_option.description += " " + desc
            else:
                current_option.description = desc

    # Don't forget the last option
    if current_option:
        options.append(current_option)

    if not options:
        return None

    has_text_option = any(opt.is_text_input for opt in options)

    return DetectedQuestion(
        header=header,
        question=question,
        options=options,
        selected_index=selected_index,
        has_text_option=has_text_option
    )


async def get_screen_text(session) -> str:
    """Get all text currently visible in the session."""
    contents = await session.async_get_screen_contents()
    lines = []
    for i in range(contents.number_of_lines):
        line = contents.line(i)
        lines.append(line.string)
    return '\n'.join(lines)


async def find_claude_session(app) -> Optional:
    """Find an active Claude session by looking for characteristic content."""
    # First try the current session
    session_id = os.environ.get('ITERM_SESSION_ID')
    if session_id and ':' in session_id:
        session_id = session_id.split(':', 1)[1]

    # We want to find a DIFFERENT session that's running Claude
    # Not the session running this script
    for window in app.terminal_windows:
        for tab in window.tabs:
            for session in tab.sessions:
                # Skip our own session
                if session.session_id == session_id:
                    continue

                try:
                    screen_text = await get_screen_text(session)
                    # Look for Claude indicators
                    if '❯' in screen_text or '☐' in screen_text:
                        return session
                except Exception:
                    continue

    return None


async def detect_question(connection, target_session_id: Optional[str] = None):
    """Detect if there's a multiple choice question displayed."""
    app = await iterm2.app.async_get_app(connection)

    session = None
    if target_session_id:
        session = app.get_session_by_id(target_session_id)
    else:
        session = await find_claude_session(app)

    if not session:
        return {"error": "no_claude_session", "message": "Could not find Claude session"}

    screen_text = await get_screen_text(session)
    clean_text = strip_ansi(screen_text)

    question = parse_multiple_choice(clean_text)

    if question:
        result = {
            "found": True,
            "session_id": session.session_id,
            "header": question.header,
            "question": question.question,
            "selected_index": question.selected_index,
            "has_text_option": question.has_text_option,
            "options": [asdict(opt) for opt in question.options]
        }
    else:
        result = {
            "found": False,
            "session_id": session.session_id,
            "raw_screen": clean_text[:2000]  # Include some screen text for debugging
        }

    return result


async def send_answer(connection, choice: int = None, text: str = None, target_session_id: str = None):
    """Send an answer to a multiple choice question.

    Args:
        choice: 1-based option number to select
        text: Custom text to type (for "Type something" option)
        target_session_id: Specific session to target
    """
    app = await iterm2.app.async_get_app(connection)

    session = None
    if target_session_id:
        session = app.get_session_by_id(target_session_id)
    else:
        session = await find_claude_session(app)

    if not session:
        return {"error": "no_claude_session", "message": "Could not find Claude session"}

    # First detect the current question to know current selection
    screen_text = await get_screen_text(session)
    clean_text = strip_ansi(screen_text)
    question = parse_multiple_choice(clean_text)

    if not question:
        return {"error": "no_question", "message": "No multiple choice question detected"}

    if text:
        # User wants to type custom text
        # First navigate to "Type something" option if it exists
        text_option_idx = None
        for i, opt in enumerate(question.options):
            if opt.is_text_input:
                text_option_idx = i
                break

        if text_option_idx is not None:
            # Navigate to text option
            moves = text_option_idx - question.selected_index
            if moves > 0:
                for _ in range(moves):
                    await session.async_send_text("\x1b[B")  # Down arrow
                    await asyncio.sleep(0.05)
            elif moves < 0:
                for _ in range(abs(moves)):
                    await session.async_send_text("\x1b[A")  # Up arrow
                    await asyncio.sleep(0.05)

            await asyncio.sleep(0.1)

        # Press Enter to select "Type something", then type the text
        await session.async_send_text("\r")
        await asyncio.sleep(0.2)
        await session.async_send_text(text)
        await asyncio.sleep(0.1)
        await session.async_send_text("\r")

        return {"success": True, "action": "typed_text", "text": text}

    elif choice is not None:
        # Navigate to the chosen option (1-based index)
        target_idx = choice - 1
        if target_idx < 0 or target_idx >= len(question.options):
            return {"error": "invalid_choice", "message": f"Choice {choice} out of range 1-{len(question.options)}"}

        moves = target_idx - question.selected_index

        if moves > 0:
            for _ in range(moves):
                await session.async_send_text("\x1b[B")  # Down arrow
                await asyncio.sleep(0.05)
        elif moves < 0:
            for _ in range(abs(moves)):
                await session.async_send_text("\x1b[A")  # Up arrow
                await asyncio.sleep(0.05)

        await asyncio.sleep(0.1)
        await session.async_send_text("\r")  # Enter to select

        selected_option = question.options[target_idx]
        return {
            "success": True,
            "action": "selected_option",
            "choice": choice,
            "label": selected_option.label
        }

    return {"error": "no_action", "message": "Specify either choice number or text"}


async def main_detect(connection, args):
    result = await detect_question(connection, args.session)
    print(json.dumps(result, indent=2 if not args.json else None))


async def main_answer(connection, args):
    result = await send_answer(
        connection,
        choice=args.choice,
        text=args.text,
        target_session_id=args.session
    )
    print(json.dumps(result, indent=2))


async def watch_for_questions(connection, args):
    """Watch for questions and output them as they appear."""
    app = await iterm2.app.async_get_app(connection)

    current_session_id = os.environ.get('ITERM_SESSION_ID', '')
    if ':' in current_session_id:
        current_session_id = current_session_id.split(':', 1)[1]

    seen_questions = set()  # Track questions we've already reported
    print(f"Watching for questions (poll interval: {args.interval}s)...", file=sys.stderr)

    while True:
        for window in app.terminal_windows:
            for tab in window.tabs:
                for session in tab.sessions:
                    if session.session_id == current_session_id:
                        continue

                    try:
                        screen_text = await get_screen_text(session)
                        clean_text = strip_ansi(screen_text)
                        question = parse_multiple_choice(clean_text)

                        if question:
                            # Create unique key for this question
                            key = f"{session.session_id}:{question.question}"
                            if key not in seen_questions:
                                seen_questions.add(key)
                                result = {
                                    "event": "question_detected",
                                    "session_id": session.session_id,
                                    "header": question.header,
                                    "question": question.question,
                                    "selected_index": question.selected_index,
                                    "has_text_option": question.has_text_option,
                                    "options": [asdict(opt) for opt in question.options]
                                }
                                print(json.dumps(result))
                                sys.stdout.flush()

                                if args.once:
                                    return
                    except Exception:
                        pass

        await asyncio.sleep(args.interval)


async def list_sessions(connection, args):
    """List all iTerm2 sessions with preview of content."""
    app = await iterm2.app.async_get_app(connection)

    current_session_id = os.environ.get('ITERM_SESSION_ID', '')
    if ':' in current_session_id:
        current_session_id = current_session_id.split(':', 1)[1]

    sessions = []
    for window in app.terminal_windows:
        for tab in window.tabs:
            for session in tab.sessions:
                try:
                    screen_text = await get_screen_text(session)
                    clean_text = strip_ansi(screen_text)

                    # Get last non-empty line as preview
                    lines = [l.strip() for l in clean_text.split('\n') if l.strip()]
                    preview = lines[-1][:80] if lines else "(empty)"

                    # Check for question
                    question = parse_multiple_choice(clean_text)

                    sessions.append({
                        "session_id": session.session_id,
                        "is_current": session.session_id == current_session_id,
                        "has_question": question is not None,
                        "question_text": question.question if question else None,
                        "preview": preview
                    })
                except Exception as e:
                    sessions.append({
                        "session_id": session.session_id,
                        "error": str(e)
                    })

    print(json.dumps(sessions, indent=2))


def main():
    parser = argparse.ArgumentParser(description="Handle Claude Code multiple choice questions")
    parser.add_argument('--session', '-s', help="Target session ID (auto-detects if not specified)")

    subparsers = parser.add_subparsers(dest='command', required=True)

    # list command
    list_parser = subparsers.add_parser('list', help='List all iTerm2 sessions')

    # watch command
    watch_parser = subparsers.add_parser('watch', help='Watch for questions and output JSON when detected')
    watch_parser.add_argument('--interval', '-i', type=float, default=1.0, help='Poll interval in seconds (default: 1.0)')
    watch_parser.add_argument('--once', action='store_true', help='Exit after first question detected')

    # detect command
    detect_parser = subparsers.add_parser('detect', help='Detect multiple choice question')
    detect_parser.add_argument('--json', '-j', action='store_true', help='Compact JSON output')

    # answer command
    answer_parser = subparsers.add_parser('answer', help='Answer a question')
    answer_parser.add_argument('choice', nargs='?', type=int, help='Option number (1-based)')
    answer_parser.add_argument('--text', '-t', help='Custom text to type')

    args = parser.parse_args()

    if args.command == 'list':
        iterm2.run_until_complete(lambda conn: list_sessions(conn, args))
    elif args.command == 'watch':
        iterm2.run_until_complete(lambda conn: watch_for_questions(conn, args))
    elif args.command == 'detect':
        iterm2.run_until_complete(lambda conn: main_detect(conn, args))
    elif args.command == 'answer':
        if args.choice is None and args.text is None:
            parser.error("answer requires either a choice number or --text")
        iterm2.run_until_complete(lambda conn: main_answer(conn, args))


if __name__ == "__main__":
    main()
