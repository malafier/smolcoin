#!/bin/bash

SESSION_NAME="p2p"

SOURCE="source .venv/bin/activate"

OPTIONS=(
    "-p 3000"
    "-p 3001 -P 127.0.0.1:3000"
    "-p 3002 -P 127.0.0.1:3000"
    "-p 3003 -P 127.0.0.1:3002"
    "-p 3004 -P 127.0.0.1:3003"
    "-p 3005 -P 127.0.0.1:3004"
)

tmux has-session -t "$SESSION_NAME" 2>/dev/null
if [ $? -eq 0 ]; then 
    tmux kill-session -t "$SESSION_NAME"
fi

tmux new-session -d -s "$SESSION_NAME" -c $(pwd)

tmux split-window -h -t "$SESSION_NAME:1.1" -c "$(pwd)"
tmux split-window -h -t "$SESSION_NAME:1.2" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:1.1" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:1.2" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:1.3" -c "$(pwd)"

for i in {1..6}; do
    sleep 1
    PANE_ID="$SESSION_NAME:1.$i"
    OPTION=${OPTIONS[$i-1]}

    tmux send-keys -t "$PANE_ID" "$SOURCE_COMMAND" C-m

    tmux send-keys -t "$PANE_ID" "uv run main.py $OPTION" C-m
done

tmux select-layout -t "$SESSION_NAME:1" tiled

