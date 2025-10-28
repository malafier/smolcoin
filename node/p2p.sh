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

tmux new-session -d -s "$SESSION_NAME" -c $(pwd)

tmux split-window -h -t "$SESSION_NAME:0.0" -c "$(pwd)"
tmux split-window -h -t "$SESSION_NAME:0.1" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:0.0" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:0.1" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:0.2" -c "$(pwd)"

PANE_INDICES=(0 1 2 3 4 5)

for i in {0..5}; do
    PANE_INDEX=${PANE_INDICES[$i]}
    PANE_ID="$SESSION_NAME:0.$PANE_INDEX"
    OPTION=${OPTIONS[$i]}

    tmux send-keys -t "$PANE_ID" "$SOURCE_COMMAND" C-m

    tmux send-keys -t "$PANE_ID" "uv run main.py $OPTION" C-m
done

tmux select-layout -t "$SESSION_NAME:0" tiled

