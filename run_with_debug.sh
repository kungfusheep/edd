#!/bin/bash

# Clear old logs
rm -f /tmp/edd_*.log

echo "Starting edd with test1.json..."
echo "Try pressing 'c' to enter connect mode, then ESC to exit, then 'q' to quit"
echo "Expected behavior: labels should restart from 'a' for visible nodes"
echo ""

# Run edd
./edd test1.json

echo ""
echo "Application closed. Checking debug logs..."

if [ -f "/tmp/edd_labels.log" ]; then
    echo "=== edd_labels.log ==="
    cat /tmp/edd_labels.log
    echo ""
else
    echo "No /tmp/edd_labels.log found"
fi

if [ -f "/tmp/edd_draw_labels.log" ]; then
    echo "=== edd_draw_labels.log ==="
    cat /tmp/edd_draw_labels.log
    echo ""
else
    echo "No /tmp/edd_draw_labels.log found"
fi