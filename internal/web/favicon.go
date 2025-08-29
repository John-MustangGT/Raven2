// internal/web/favicon.go
package web

import (
    "encoding/base64"
    "net/http"
    
    "github.com/gin-gonic/gin"
)

// Embedded favicon as base64 - this is a simple crow/raven icon in SVG format
const faviconSVG = `
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
</svg>`

// Convert SVG to PNG-like ICO format (simplified approach)
// In a real implementation, you might want to use a proper ICO generation library
func (s *Server) serveFavicon(c *gin.Context) {
    // For now, serve as SVG with proper headers for favicon
    c.Header("Content-Type", "image/svg+xml")
    c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year
    c.String(http.StatusOK, faviconSVG)
}

// Alternative: serve a simple ICO favicon
func (s *Server) serveFaviconICO(c *gin.Context) {
    // This is a minimal 16x16 ICO file encoded as base64
    // Generated from the SVG above, but simplified for ICO format
    icoData := `AAABAAEAEBAAAAEAIABoBAAAFgAAACgAAAAQAAAAIAAAAAEAIAAAAAAAAAQAABMLAAATCwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA59/EAOW8xQDz4tUA594oAOf2LwDn9y8A59YrAOTBxQDnv8YAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAOfeyADxyLcA+OvZAPz23wD89t8A/PbfAPz23wD469kA8ci3AOjezAAAAAAAAAAAAAAAAAAAAAAA6uLIAPHGrgD57+EA/PbfAPz23wD89t8A/PbfAPz23wD89t8A+e/hAPHGrgDq4sgAAAAAAOfdxgDxw6oA+e3bAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A+e3bAPHDqgDn3cYA5L7CAPbj0AD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPbj0ADkvMQA7dOyAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wDs0rAA6c/BAPz23wD89t8A/PbfAOXCvADlwrwA/PbfAPz23wDlwrwA5cK8APz23wD89t8A/PbfAPz23wDpz8EA5cC5APz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A5cC5AOXG9QD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A5cb1AOvSyQD89t8A/PbfAPz23wDwrYAA8K2AAPz23wD89t8A8K2AAPCtiAD89t8A/PbfAPz23wD89t8A69LJAOXD7QD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A5cPtAOXBvAD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A5cG8AOfdyAD36tYA/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPz23wD89t8A/PbfAPfq1gDn3cgAAAAAAOfezADxyLcA+OvZAPz23wD89t8A/PbfAPz23wD469kA8ci3AOjezAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA59/EAOW8xQDz4tUA594oAOf2LwDn9y8A59YrAOTBxQDnv8YAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAP//AAD//wAA//8AAP//AAD//wAA//8AAP//AAD//wAA//8AAP//AAD//wAA//8AAP//AAD//wAA//8AAP//AAA=`
    
    // Decode the base64 ICO data
    icoBytes, err := base64.StdEncoding.DecodeString(icoData)
    if err != nil {
        // Fallback to SVG
        s.serveFavicon(c)
        return
    }
    
    c.Header("Content-Type", "image/x-icon")
    c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year
    c.Data(http.StatusOK, "image/x-icon", icoBytes)
}
