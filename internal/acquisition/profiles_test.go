package acquisition

import (
	"testing"
)

func TestSelectToolsForProfileIncludesCommonAndSelectedProfile(t *testing.T) {
	tools := []Tool{
		{
			Name:     "common-tool",
			Profiles: []string{CommonProfile},
		},
		{
			Name:     "k8s-tool",
			Profiles: []string{"k8s"},
		},
		{
			Name:     "k8s-common-tool",
			Profiles: []string{CommonProfile, "k8s"},
		},
		{
			Name:     "security-tool",
			Profiles: []string{"security"},
		},
		{
			Name: "unprofiled-tool",
		},
	}

	selected, err := SelectToolsForProfile(tools, "k8s")
	if err != nil {
		t.Fatalf("SelectToolsForProfile() error = %v", err)
	}

	if len(selected) != 3 {
		t.Fatalf(
			"SelectToolsForProfile() returned %d tools, want 3",
			len(selected),
		)
	}

	want := []string{
		"common-tool",
		"k8s-tool",
		"k8s-common-tool",
	}

	for i, tool := range selected {
		if tool.Name != want[i] {
			t.Fatalf(
				"selected[%d].Name = %q, want %q",
				i,
				tool.Name,
				want[i],
			)
		}
	}
}

func TestSelectToolsForProfileExcludesOtherProfiles(t *testing.T) {
	tools := []Tool{
		{
			Name:     "common-tool",
			Profiles: []string{CommonProfile},
		},
		{
			Name:     "k8s-tool",
			Profiles: []string{"k8s"},
		},
		{
			Name:     "security-tool",
			Profiles: []string{"security"},
		},
	}

	selected, err := SelectToolsForProfile(tools, "security")
	if err != nil {
		t.Fatalf("SelectToolsForProfile() error = %v", err)
	}

	if len(selected) != 2 {
		t.Fatalf(
			"SelectToolsForProfile() returned %d tools, want 2",
			len(selected),
		)
	}

	if selected[0].Name != "common-tool" {
		t.Fatalf(
			"selected[0].Name = %q, want %q",
			selected[0].Name,
			"common-tool",
		)
	}

	if selected[1].Name != "security-tool" {
		t.Fatalf(
			"selected[1].Name = %q, want %q",
			selected[1].Name,
			"security-tool",
		)
	}
}

func TestSelectToolsForProfilePreservesInputOrder(t *testing.T) {
	tools := []Tool{
		{
			Name:     "third",
			Profiles: []string{"k8s"},
		},
		{
			Name:     "first",
			Profiles: []string{CommonProfile},
		},
		{
			Name:     "second",
			Profiles: []string{"k8s"},
		},
	}

	selected, err := SelectToolsForProfile(tools, "k8s")
	if err != nil {
		t.Fatalf("SelectToolsForProfile() error = %v", err)
	}

	want := []string{
		"third",
		"first",
		"second",
	}

	for i, tool := range selected {
		if tool.Name != want[i] {
			t.Fatalf(
				"selected[%d].Name = %q, want %q",
				i,
				tool.Name,
				want[i],
			)
		}
	}
}

func TestSelectToolsForProfileRejectsEmptyProfile(t *testing.T) {
	_, err := SelectToolsForProfile(
		[]Tool{
			{
				Name:     "common-tool",
				Profiles: []string{CommonProfile},
			},
		},
		"",
	)
	if err == nil {
		t.Fatal("SelectToolsForProfile() error = nil, want empty-profile error")
	}
}

func TestSelectToolsForProfileRejectsCommonProfile(t *testing.T) {
	_, err := SelectToolsForProfile(
		[]Tool{
			{
				Name:     "common-tool",
				Profiles: []string{CommonProfile},
			},
		},
		CommonProfile,
	)
	if err == nil {
		t.Fatal("SelectToolsForProfile() error = nil, want reserved-profile error")
	}
}
