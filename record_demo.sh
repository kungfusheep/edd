#!/bin/bash

echo "Recording EDD demo..."
echo "This will create an asciinema recording that can be converted to GIF"
echo ""
echo "Prerequisites:"
echo "  1. Install asciinema: brew install asciinema"
echo "  2. Install svg-term-cli: npm install -g svg-term-cli"
echo "  3. Install imagemagick: brew install imagemagick"
echo ""
echo "Press Enter to start recording..."
read

# Record with asciinema
echo "Recording demo..."
asciinema rec --overwrite demo.cast -c "./edd -demo -min-delay 50 -max-delay 200 < showcase_demo.txt"

echo "Recording complete!"
echo ""
echo "To convert to GIF:"
echo "  1. Convert to SVG: svg-term --in demo.cast --out demo.svg --window"
echo "  2. Convert to GIF: convert -density 150 demo.svg demo.gif"
echo ""
echo "Or upload to asciinema.org:"
echo "  asciinema upload demo.cast"