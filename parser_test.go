package main

import (
	"testing"
)

func TestParseRepositoriesWithSlashes(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []Repository
	}{
		{
			name:  "Single repo with branch containing slash",
			input: []string{"microsoft/vscode:feat/rebrand-main"},
			expected: []Repository{
				{
					Owner:              "microsoft",
					Name:               "vscode",
					ProductionBranches: []string{"feat/rebrand-main"},
				},
			},
		},
		{
			name:  "Multiple branches with slashes",
			input: []string{"microsoft/vscode:main,feat/rebrand-main,release/v2.0"},
			expected: []Repository{
				{
					Owner:              "microsoft",
					Name:               "vscode",
					ProductionBranches: []string{"main", "feat/rebrand-main", "release/v2.0"},
				},
			},
		},
		{
			name:  "Complex branch names",
			input: []string{"owner/repo:feature/ui/new-design,hotfix/security/patch"},
			expected: []Repository{
				{
					Owner:              "owner",
					Name:               "repo",
					ProductionBranches: []string{"feature/ui/new-design", "hotfix/security/patch"},
				},
			},
		},
		{
			name:  "Default branch when no colon",
			input: []string{"microsoft/vscode"},
			expected: []Repository{
				{
					Owner:              "microsoft",
					Name:               "vscode",
					ProductionBranches: []string{"main"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRepositories(tt.input)
			if err != nil {
				t.Fatalf("parseRepositories() error = %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("Expected %d repos, got %d", len(tt.expected), len(result))
			}

			for i, repo := range result {
				expected := tt.expected[i]

				if repo.Owner != expected.Owner {
					t.Errorf("Expected Owner %s, got %s", expected.Owner, repo.Owner)
				}

				if repo.Name != expected.Name {
					t.Errorf("Expected Name %s, got %s", expected.Name, repo.Name)
				}

				if len(repo.ProductionBranches) != len(expected.ProductionBranches) {
					t.Errorf("Expected %d branches, got %d", len(expected.ProductionBranches), len(repo.ProductionBranches))
				}

				for j, branch := range repo.ProductionBranches {
					if branch != expected.ProductionBranches[j] {
						t.Errorf("Expected branch %s, got %s", expected.ProductionBranches[j], branch)
					}
				}
			}
		})
	}
}
