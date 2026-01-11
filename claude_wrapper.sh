#!/bin/bash
# Wrapper to launch claude with proper PATH and skip permission prompts
# Used by get_claude_status.py for automated status checking

# Allow override via environment variable
if [ -n "$RCODEGEN_NODE_PATH" ]; then
    export PATH="$RCODEGEN_NODE_PATH:$PATH"
elif [ -d "$HOME/.nvm/versions/node" ]; then
    # nvm - use latest installed version
    latest=$(ls -v "$HOME/.nvm/versions/node" 2>/dev/null | tail -1)
    if [ -n "$latest" ]; then
        export PATH="$HOME/.nvm/versions/node/$latest/bin:$PATH"
    fi
elif [ -d "/opt/homebrew/bin" ]; then
    # Homebrew on Apple Silicon
    export PATH="/opt/homebrew/bin:$PATH"
elif [ -d "/usr/local/bin" ]; then
    # Homebrew on Intel Mac / standard location
    export PATH="/usr/local/bin:$PATH"
fi

# Verify claude is available
if ! command -v claude &> /dev/null; then
    echo "Error: claude not found in PATH" >&2
    echo "Set RCODEGEN_NODE_PATH to the directory containing claude" >&2
    exit 1
fi

exec claude --dangerously-skip-permissions "$@"
