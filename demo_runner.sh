#!/bin/bash

echo "=================================================="
echo "       EDD - Fast Terminal Diagram Editor        "
echo "=================================================="
echo ""
echo "Demo 1: Create a flowchart in 15 seconds"
echo "-----------------------------------------"
echo ""
echo "Commands:"
echo "  i - Insert mode (add nodes)"
echo "  c - Connect nodes"
echo "  :w - Save diagram"
echo ""
echo "Input sequence:"
cat quick_demo.txt | sed 's/^/  /'
echo ""
echo "Press Enter to start..."
read

echo "Running demo..."
./edd -demo -min-delay 50 -max-delay 150 -line-delay 300 < quick_demo.txt

echo ""
echo "Demo 2: Create a sequence diagram"
echo "----------------------------------"
echo ""
echo "Commands:"
echo "  ts - Switch to sequence diagram"
echo "  i - Insert participants"
echo "  c - Connect with messages"
echo ""
echo "Input sequence:"
cat sequence_demo.txt | sed 's/^/  /'
echo ""
echo "Press Enter to start..."
read

echo "Running demo..."
./edd -demo -min-delay 50 -max-delay 150 -line-delay 300 < sequence_demo.txt

echo ""
echo "✨ Diagrams created in seconds!"
echo ""
echo "Key features demonstrated:"
echo "  • Fast keyboard-driven interface"
echo "  • Multiple diagram types (flowchart, sequence)"
echo "  • Instant visual feedback"
echo "  • JSON export for version control"