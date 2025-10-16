#!/bin/bash

# Script to create a demo GIF for edd
echo "==================================="
echo "     EDD Demo GIF Creator"
echo "==================================="
echo ""

# Check for dependencies
check_dependency() {
    if ! command -v $1 &> /dev/null; then
        echo "❌ $1 is not installed"
        echo "   Install with: $2"
        return 1
    else
        echo "✅ $1 is installed"
        return 0
    fi
}

echo "Checking dependencies..."
check_dependency "asciinema" "brew install asciinema"
ASCIINEMA=$?

check_dependency "svg-term" "npm install -g svg-term-cli"
SVGTERM=$?

check_dependency "convert" "brew install imagemagick"
IMAGEMAGICK=$?

if [ $ASCIINEMA -ne 0 ] || [ $SVGTERM -ne 0 ] || [ $IMAGEMAGICK -ne 0 ]; then
    echo ""
    echo "Please install missing dependencies and try again."
    exit 1
fi

echo ""
echo "All dependencies installed!"
echo ""

# Create a clean demo script
cat > clean_demo.txt << 'EOF'
ts
iUser
API Gateway
Auth Service
Database

cUser
API Gateway
POST /login
cAPI Gateway
Auth Service
Validate
cAuth Service
Database
Query user
cDatabase
Auth Service
User data
jd
hug
je
hur
:w demo.json
:q
EOF

echo "Starting demo recording..."
echo "The demo will run automatically with realistic typing delays."
echo ""

# Record with asciinema
asciinema rec \
    --overwrite \
    --title "EDD - Terminal Diagram Editor" \
    --idle-time-limit 2 \
    demo.cast \
    --command "./edd -demo -min-delay 50 -max-delay 150 -line-delay 300 < clean_demo.txt"

echo ""
echo "Recording complete! Converting to GIF..."

# Convert to SVG with nice styling
svg-term \
    --in demo.cast \
    --out demo.svg \
    --window \
    --no-cursor \
    --width 80 \
    --height 24

# Convert SVG to high-quality GIF
convert \
    -density 200 \
    -delay 5 \
    -loop 0 \
    -background "#1e1e1e" \
    demo.svg \
    demo.gif

# Optimize the GIF
if command -v gifsicle &> /dev/null; then
    echo "Optimizing GIF size..."
    gifsicle -O3 --colors 64 demo.gif -o demo_optimized.gif
    mv demo_optimized.gif demo.gif
fi

echo ""
echo "✨ Demo GIF created successfully!"
echo ""
echo "Files created:"
echo "  - demo.cast (asciinema recording)"
echo "  - demo.svg (SVG version)"
echo "  - demo.gif (Final GIF)"
echo ""
echo "You can also upload to asciinema.org:"
echo "  asciinema upload demo.cast"
echo ""
echo "To add to README:"
echo "  ![EDD Demo](demo.gif)"

# Clean up
rm clean_demo.txt