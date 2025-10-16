#!/bin/bash

# Script to verify activation deletion works properly

echo "Creating test diagram with activations..."

cat > test_activation_verify.mmd << 'EOF'
sequenceDiagram
    participant A as Alice
    participant B as Bob

    A->>+B: Start work
    B->>B: Process
    B->>-A: Done

    A->>+B: Another task
    B->>-A: Completed
EOF

echo "Test diagram created with existing activations."
echo ""
echo "Instructions to test activation deletion:"
echo "1. The diagram should already have activation boxes (thick bars) on participant B"
echo "2. Press 'V' (Shift+V) to enter DELETE ACTIVATION mode"
echo "3. You should see labels ONLY on connections that have activation hints"
echo "4. Select any labeled connection to delete the entire activation span"
echo "5. The activation box should disappear completely"
echo ""
echo "Starting edd..."

./edd test_activation_verify.mmd