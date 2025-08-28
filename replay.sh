#!/bin/bash

# replay.sh - Replay commands to edd with randomized natural timing
# Usage: ./replay.sh commands.txt | ./edd

if [ $# -ne 1 ]; then
    echo "Usage: $0 commands.txt"
    echo "Example commands.txt:"
    echo "  aNode1"
    echo "  aNode2" 
    echo "  cab"
    exit 1
fi

# Read each line and output characters with random delays
while IFS= read -r line; do
    for (( i=0; i<${#line}; i++ )); do
        char="${line:$i:1}"
        printf "%s" "$char"
        
        # Random delay between 30-80ms per character
        sleep 0.0$(( RANDOM % 5 + 3 ))
    done
    
    # Newline with slightly longer pause
    printf "\n"
    sleep 0.$(( RANDOM % 5 + 3 ))
done < "$1"