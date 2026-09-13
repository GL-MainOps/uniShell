package acquisition

import (
	"fmt"
	"strings"
)

const CommonProfile = "common"

func SelectToolsForProfile(tools []Tool, profile string) ([]Tool, error) {
	profile = strings.TrimSpace(profile)

	if profile == "" {
		return nil, fmt.Errorf("profile name is required")
	}

	if profile == CommonProfile {
		return nil, fmt.Errorf(
			"profile %q is reserved and cannot be selected directly",
			CommonProfile,
		)
	}

	if !hasProfileInTools(tools, profile) {
		return nil, fmt.Errorf("unknown profile %q", profile)
	}

	selected := make([]Tool, 0, len(tools))

	for _, tool := range tools {
		if hasProfile(tool.Profiles, CommonProfile) ||
			hasProfile(tool.Profiles, profile) {
			selected = append(selected, tool)
		}
	}

	return selected, nil
}

func hasProfileInTools(tools []Tool, profile string) bool {
	for _, tool := range tools {
		if hasProfile(tool.Profiles, profile) {
			return true
		}
	}

	return false
}

func hasProfile(profiles []string, profile string) bool {
	for _, candidate := range profiles {
		if candidate == profile {
			return true
		}
	}

	return false
}
