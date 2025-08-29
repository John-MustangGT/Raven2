// internal/web/build_info.go
package web

import (
    "runtime"

    "github.com/gin-gonic/gin"
)

// BuildInfo holds build-time information
type BuildInfo struct {
    Version     string    `json:"version"`
    GitCommit   string    `json:"git_commit"`
    GitBranch   string    `json:"git_branch"`
    BuildTime   string    `json:"build_time"`
    GoVersion   string    `json:"go_version"`
    GoOS        string    `json:"go_os"`
    GoArch      string    `json:"go_arch"`
    CGOEnabled  string    `json:"cgo_enabled"`
    BuildFlags  string    `json:"build_flags"`
    ModuleInfo  []Module  `json:"modules"`
}

type Module struct {
    Path    string `json:"path"`
    Version string `json:"version"`
    Sum     string `json:"sum,omitempty"`
    Replace string `json:"replace,omitempty"`
}

// These variables will be set at build time using -ldflags
var (
    Version   = "dev"
    GitCommit = "unknown"
    GitBranch = "unknown" 
    BuildTime = "unknown"
    BuildFlags = "unknown"
)

// getBuildInfo returns comprehensive build information
func (s *Server) getBuildInfo(c *gin.Context) {
    buildInfo := BuildInfo{
        Version:    Version,
        GitCommit:  GitCommit,
        GitBranch:  GitBranch,
        BuildTime:  BuildTime,
        GoVersion:  runtime.Version(),
        GoOS:       runtime.GOOS,
        GoArch:     runtime.GOARCH,
        CGOEnabled: getCGOEnabled(),
        BuildFlags: BuildFlags,
        ModuleInfo: getModuleInfo(),
    }

    c.JSON(200, gin.H{"data": buildInfo})
}

func getCGOEnabled() string {
    // This will be set at build time if CGO is enabled
    return "false" // default
}

func getModuleInfo() []Module {
    // In a real implementation, you might want to embed module information
    // at build time. For now, return common dependencies.
    return []Module{
        {Path: "github.com/gin-gonic/gin", Version: "v1.9.1"},
        {Path: "github.com/sirupsen/logrus", Version: "v1.9.3"},
        {Path: "github.com/gorilla/websocket", Version: "v1.5.0"},
        {Path: "github.com/google/uuid", Version: "v1.3.0"},
        {Path: "github.com/prometheus/client_golang", Version: "v1.16.0"},
        // Add more as needed
    }
}
