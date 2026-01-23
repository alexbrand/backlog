package cli

import (
	"fmt"
	"os"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/config"
	"github.com/alexbrand/backlog/internal/github"
	"github.com/alexbrand/backlog/internal/local"
)

// getBackendAndConfig returns the appropriate backend and configuration based on
// the workspace settings. If no config is found, it falls back to checking for
// a local .backlog directory.
func getBackendAndConfig() (backend.Backend, backend.Config, *config.Workspace, error) {
	var b backend.Backend
	var backendCfg backend.Config
	var ws *config.Workspace

	// Try to get workspace from config
	workspace, _, err := config.GetWorkspace(GetWorkspace())
	if err == nil {
		ws = workspace
		// Have config - use it
		b, err = backend.Get(ws.Backend)
		if err != nil {
			return nil, backend.Config{}, nil, err
		}

		cfg := config.Get()
		backendCfg = backend.Config{
			AgentID:          ResolveAgentID(ws),
			AgentLabelPrefix: ws.AgentLabelPrefix,
		}

		switch ws.Backend {
		case "local":
			path := ws.Path
			if path == "" {
				path = ".backlog"
			}
			backendCfg.Workspace = &local.WorkspaceConfig{
				Path:     path,
				LockMode: local.LockMode(ws.LockMode),
				GitSync:  ws.GitSync,
			}
		case "github":
			backendCfg.Workspace = &github.WorkspaceConfig{
				Repo:        ws.Repo,
				Project:     ws.Project,
				StatusField: ws.StatusField,
				StatusMap:   convertStatusMap(ws.StatusMap),
			}
			// AgentID is already set above via ResolveAgentID
			if cfg != nil && cfg.Defaults.AgentID != "" && backendCfg.AgentID == "" {
				backendCfg.AgentID = cfg.Defaults.AgentID
			}
		default:
			return nil, backend.Config{}, nil, fmt.Errorf("unsupported backend: %s", ws.Backend)
		}
	} else {
		// No config - check for local .backlog directory
		if _, statErr := os.Stat(".backlog"); statErr == nil {
			// Local .backlog directory exists - use local backend
			b, err = backend.Get("local")
			if err != nil {
				return nil, backend.Config{}, nil, err
			}
			backendCfg = backend.Config{
				Workspace: &local.WorkspaceConfig{Path: ".backlog"},
			}
		} else {
			// No config and no local .backlog directory
			return nil, backend.Config{}, nil, err
		}
	}

	return b, backendCfg, ws, nil
}

// convertStatusMap converts the config.Status map to github.StatusMapping map.
func convertStatusMap(statusMap map[string]config.Status) map[backend.Status]github.StatusMapping {
	if statusMap == nil {
		return nil
	}

	result := make(map[backend.Status]github.StatusMapping)
	for status, mapping := range statusMap {
		result[backend.Status(status)] = github.StatusMapping{
			State:  mapping.State,
			Labels: mapping.Labels,
		}
	}
	return result
}

// connectBackend is a convenience function that gets the backend and connects to it.
// It returns the backend, workspace config, and a cleanup function to disconnect.
func connectBackend() (backend.Backend, *config.Workspace, func(), error) {
	b, backendCfg, ws, err := getBackendAndConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	if err := b.Connect(backendCfg); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to backend: %w", err)
	}

	cleanup := func() {
		b.Disconnect()
	}

	return b, ws, cleanup, nil
}
