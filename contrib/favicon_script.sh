#!/bin/bash

# Create a favicon.ico file for Raven
# This script creates a simple crow/raven icon in ICO format

# Method 1: Create favicon.ico using base64 data
cat > web/favicon.ico << 'EOF'
AAABAAEAEBAAAAEAIABoBAAAFgAAACgAAAAQAAAAIAAAAAEAIAAAAAAAAAQAABMLAAATCwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA59/EAOW8xQDz4tUA594oAOf2LwDn9y8A59YrAOTBxQDnv8YAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA59/EAOW8xQDz4tUA594oAOf2LwDn9y8A59YrAOTBxQDnv8YAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA4uLIAPHGrgD57+EA/PbfAPz23wD89t8A/PbfAPz23wD89t8A+e/hAPHGrgDq4sgAAAAAAAAAAAAAAAAA6uLIAPHGrgD57+EA/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPnv4QDxxq4A6uLIAAAAAAAAAAAAAAAA59/EAOW8xQDz4tUA594oAOf2LwDn9y8A59YrAOTBxQDnv8YAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAfveyAD57+EA/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A+e/hAPHGrgA6+PIAAAAAAAAAAAAAwOTVAO59z+D3v4QA/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wDs0rAA7dOyAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wDs0rAAAA49MzBAPz23wD89t8A/PbfAOXCvADlwrwA/PbfAPz23wDlwrwA5cK8APz23wD89t8A/PbfAPz23wDpz8EA5cC5APz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A5cC5AOXG9QD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A5cb1AOvSyQD89t8A/PbfAPz23wDwrYAA8K2AAPz23wD89t8A8K2AAPCtiAD89t8A/PbfAPz23wD89t8A69LJAOXD7QD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A5cPtAOXBvAD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A5cG8AOfdyAD36tYA/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPfq1gDn3cgAAAAAAAAAAAAAAAAA59/EAOW8xQDz4tUA594oAOf2LwDn9y8A59YrAOTBxQDnv8YAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP//AAD//wAA//8AAP//AAD//wAA//8AAP//AAD//wAA//8AAP//AAD//8AAP//AAD//wAA//8AAP//AAD//wAA
EOF

echo "favicon.ico created in web/ directory"

# Method 2: Create using ImageMagick (if available)
if command -v convert &> /dev/null; then
    echo "ImageMagick found, creating high-quality favicon..."
    
    # Create a simple SVG first
    cat > /tmp/raven-icon.svg << 'SVGEOF'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32">
  <defs>
    <linearGradient id="bg" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#3b82f6;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#1d4ed8;stop-opacity:1" />
    </linearGradient>
  </defs>
  
  <!-- Background circle -->
  <circle cx="16" cy="16" r="16" fill="url(#bg)"/>
  
  <!-- Raven/Crow silhouette -->
  <g fill="#ffffff" transform="translate(4,6)">
    <!-- Body -->
    <ellipse cx="12" cy="14" rx="8" ry="6"/>
    
    <!-- Head -->
    <circle cx="8" cy="8" r="5"/>
    
    <!-- Beak -->
    <polygon points="3,8 8,6 8,10"/>
    
    <!-- Eye -->
    <circle cx="10" cy="7" r="1" fill="#3b82f6"/>
    
    <!-- Wing detail -->
    <path d="M 12,10 Q 18,8 20,12 Q 18,16 12,16 Z" opacity="0.7"/>
    
    <!-- Tail -->
    <path d="M 20,14 Q 24,12 24,18 Q 20,20 18,16 Z"/>
  </g>
</svg>
SVGEOF

    # Convert SVG to ICO using ImageMagick
    convert -background transparent /tmp/raven-icon.svg \
            \( -clone 0 -resize 16x16 \) \
            \( -clone 0 -resize 32x32 \) \
            \( -clone 0 -resize 48x48 \) \
            \( -clone 0 -resize 64x64 \) \
            -delete 0 web/favicon.ico
    
    # Also create PNG versions
    convert -background transparent /tmp/raven-icon.svg -resize 32x32 web/favicon-32x32.png
    convert -background transparent /tmp/raven-icon.svg -resize 16x16 web/favicon-16x16.png
    
    echo "High-quality favicon created with ImageMagick"
    
    # Clean up
    rm -f /tmp/raven-icon.svg
else
    echo "ImageMagick not found, using base64 encoded favicon"
fi

# Create a simple SVG favicon as well
cat > web/favicon.svg << 'SVGEOF'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32">
  <defs>
    <linearGradient id="bg" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#3b82f6;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#1d4ed8;stop-opacity:1" />
    </linearGradient>
  </defs>
  
  <!-- Background circle -->
  <circle cx="16" cy="16" r="16" fill="url(#bg)"/>
  
  <!-- Raven/Crow silhouette -->
  <g fill="#ffffff" transform="translate(4,6)">
    <!-- Body -->
    <ellipse cx="12" cy="14" rx="8" ry="6"/>
    
    <!-- Head -->
    <circle cx="8" cy="8" r="5"/>
    
    <!-- Beak -->
    <polygon points="3,8 8,6 8,10"/>
    
    <!-- Eye -->
    <circle cx="10" cy="7" r="1" fill="#3b82f6"/>
    
    <!-- Wing detail -->
    <path d="M 12,10 Q 18,8 20,12 Q 18,16 12,16 Z" opacity="0.7"/>
    
    <!-- Tail -->
    <path d="M 20,14 Q 24,12 24,18 Q 20,20 18,16 Z"/>
  </g>
</svg>
SVGEOF

echo "favicon.svg created in web/ directory"

# Make the script executable
chmod +x "$0"

echo ""
echo "Favicon creation complete!"
echo "Files created:"
echo "  web/favicon.ico    - ICO format for broad compatibility"
echo "  web/favicon.svg    - SVG format for modern browsers"
if command -v convert &> /dev/null; then
    echo "  web/favicon-32x32.png - 32x32 PNG"
    echo "  web/favicon-16x16.png - 16x16 PNG"
fi
echo ""
echo "The favicon features a stylized raven/crow in Raven's brand colors"
echo "and should now appear in browser tabs and bookmarks."
