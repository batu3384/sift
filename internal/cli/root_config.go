package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/engine"
)

func (r *runtimeState) persistConfig(cfg config.Config) error {
	cfg = config.Normalize(cfg)
	if err := config.Save(cfg); err != nil {
		return err
	}
	r.cfg = cfg
	r.service = engine.NewService(cfg, r.store)
	return nil
}

func parseCommandWhitelistArgs(args []string) (action string, path string, err error) {
	switch len(args) {
	case 0:
		return "list", "", nil
	case 1:
		if strings.EqualFold(args[0], "list") {
			return "list", "", nil
		}
		return "", "", fmt.Errorf("use --whitelist list, --whitelist add <path>, or --whitelist remove <path>")
	case 2:
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "add":
			if strings.TrimSpace(args[1]) == "" {
				return "", "", fmt.Errorf("path cannot be empty")
			}
			return "add", args[1], nil
		case "remove":
			if strings.TrimSpace(args[1]) == "" {
				return "", "", fmt.Errorf("path cannot be empty")
			}
			return "remove", args[1], nil
		default:
			return "", "", fmt.Errorf("unsupported whitelist action %q", args[0])
		}
	default:
		return "", "", fmt.Errorf("use --whitelist list, --whitelist add <path>, or --whitelist remove <path>")
	}
}

func normalizeUpdateChannelFlag(value string) (engine.UpdateChannel, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(engine.UpdateChannelStable):
		return engine.UpdateChannelStable, nil
	case string(engine.UpdateChannelNightly):
		return engine.UpdateChannelNightly, nil
	default:
		return "", fmt.Errorf("unsupported update channel %q", value)
	}
}

func (r *runtimeState) handleCommandWhitelist(cmd *cobra.Command, command string, args []string) error {
	action, path, err := parseCommandWhitelistArgs(args)
	if err != nil {
		return err
	}
	command = config.NormalizeCommandName(command)
	scopes := config.Normalize(r.cfg).CommandExcludes
	switch action {
	case "list":
		if r.flags.JSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"command":          command,
				"paths":            scopes[command],
				"command_excludes": scopes,
			})
		}
		if len(scopes[command]) == 0 {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "No exclusions configured for %s.\n", command)
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", command)
		for _, item := range scopes[command] {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", item)
		}
		return nil
	case "add":
		cfg, normalizedCommand, normalizedPath, err := config.AddCommandExclude(r.cfg, command, path)
		if err != nil {
			return err
		}
		if err := r.persistConfig(cfg); err != nil {
			return err
		}
		if r.flags.JSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"command":          normalizedCommand,
				"path":             normalizedPath,
				"command_excludes": r.cfg.CommandExcludes,
			})
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s exclusion added: %s\n", normalizedCommand, normalizedPath)
		return err
	case "remove":
		cfg, normalizedCommand, normalizedPath, removed, err := config.RemoveCommandExclude(r.cfg, command, path)
		if err != nil {
			return err
		}
		if removed {
			if err := r.persistConfig(cfg); err != nil {
				return err
			}
		}
		if r.flags.JSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
				"command":          normalizedCommand,
				"path":             normalizedPath,
				"removed":          removed,
				"command_excludes": r.cfg.CommandExcludes,
			})
		}
		if !removed {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s exclusion not found: %s\n", normalizedCommand, normalizedPath)
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s exclusion removed: %s\n", normalizedCommand, normalizedPath)
		return err
	default:
		return fmt.Errorf("unsupported whitelist action %q", action)
	}
}
