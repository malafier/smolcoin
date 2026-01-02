#!/bin/bash

SESSION_NAME="p2p"

WIN1=(
    "-p 3000 -M"
    "-p 3001 -P 127.0.0.1:3000"
    "-p 3002 -P 127.0.0.1:3000"
    "-p 3003 -P 127.0.0.1:3002"
    "-p 3004 -P 127.0.0.1:3003"
    "-p 3005 -P 127.0.0.1:3004"
)

WIN2=(
    "-p 3006 -M"
    "-p 3007 -P 127.0.0.1:3006"
    "-p 3008 -P 127.0.0.1:3007"
    "-p 3009"
)

tmux has-session -t "$SESSION_NAME" 2>/dev/null
if [ $? -eq 0 ]; then 
    tmux kill-session -t "$SESSION_NAME"
fi


# WINDOW 1
tmux new-session -d -s "$SESSION_NAME" -n "Main Network" -c "$(pwd)"

tmux split-window -h -t "$SESSION_NAME:1.1" -c "$(pwd)"
tmux split-window -h -t "$SESSION_NAME:1.2" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:1.1" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:1.2" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:1.3" -c "$(pwd)"

tmux select-layout -t "$SESSION_NAME:1" tiled
for i in "${!WIN1[@]}"; do
    sleep 1
    PANE_ID="$SESSION_NAME:1.$(($i+1))"
    OPTION=${WIN1[$i]}
    tmux send-keys -t "$PANE_ID" "go run ./main.go $OPTION" C-m
done

# WINDOW 2
tmux new-window -t "$SESSION_NAME" -n "Secondary Network" -c "$(pwd)"

tmux split-window -h -t "$SESSION_NAME:2.1" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:2.1" -c "$(pwd)"
tmux split-window -v -t "$SESSION_NAME:2.2" -c "$(pwd)"

tmux select-layout -t "$SESSION_NAME:2" tiled
for i in "${!WIN2[@]}"; do
    sleep 1
    PANE_ID="$SESSION_NAME:2.$(($i+1))"
    OPTION=${WIN2[$i]}
    tmux send-keys -t "$PANE_ID" "go run ./main.go $OPTION" C-m
done


# Attach
tmux select-window -t "$SESSION_NAME:1"
tmux attach
