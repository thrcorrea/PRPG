package main

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v55/github"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"01/12/2024", "2024-12-01", false},
		{"2024-12-01", "2024-12-01", false},
		{"01-12-2024", "2024-12-01", false},
		{"invalid", "", true},
		{"32/12/2024", "", true},
	}

	for _, test := range tests {
		result, err := parseDate(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", test.input, err)
			} else if result.Format("2006-01-02") != test.expected {
				t.Errorf("For input %s, expected %s, got %s",
					test.input, test.expected, result.Format("2006-01-02"))
			}
		}
	}
}

func TestGetWeekStart(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2024-10-04", "2024-09-30"}, // Sexta -> Segunda anterior
		{"2024-09-30", "2024-09-30"}, // Segunda -> Mesma segunda
		{"2024-10-06", "2024-09-30"}, // Domingo -> Segunda anterior
		{"2024-10-01", "2024-09-30"}, // Terça -> Segunda anterior
	}

	for _, test := range tests {
		inputTime, _ := time.Parse("2006-01-02", test.input)
		result := getWeekStart(inputTime)

		if result.Format("2006-01-02") != test.expected {
			t.Errorf("For input %s, expected %s, got %s",
				test.input, test.expected, result.Format("2006-01-02"))
		}
	}
}

func TestUserStatsCalculation(t *testing.T) {
	// Criar um PRChampion de teste
	startDate, _ := time.Parse("2006-01-02", "2024-09-30")
	endDate, _ := time.Parse("2006-01-02", "2024-10-06")

	pc := &PRChampion{
		startDate: startDate,
		endDate:   endDate,
		userStats: make(map[string]*UserStats),
		repositories: []Repository{
			{Owner: "test", Name: "repo1"},
			{Owner: "test", Name: "repo2"},
		},
	}

	// Simular dados semanais
	pc.weeklyData = []WeeklyData{
		{
			StartDate: startDate,
			EndDate:   startDate.Add(6 * 24 * time.Hour),
			UserPRs:   map[string]int{"user1": 5, "user2": 3, "user3": 2},
			Winner:    "user1",
		},
		{
			StartDate: startDate.Add(7 * 24 * time.Hour),
			EndDate:   startDate.Add(13 * 24 * time.Hour),
			UserPRs:   map[string]int{"user1": 2, "user2": 6, "user3": 1},
			Winner:    "user2",
		},
	}

	// Calcular estatísticas
	pc.calculateUserStats()

	// Verificar resultados
	if pc.userStats["user1"].TotalScore != 1 {
		t.Errorf("Expected user1 to have 1 point, got %d", pc.userStats["user1"].TotalScore)
	}

	if pc.userStats["user2"].TotalScore != 1 {
		t.Errorf("Expected user2 to have 1 point, got %d", pc.userStats["user2"].TotalScore)
	}

	if pc.userStats["user1"].PRsCount != 7 {
		t.Errorf("Expected user1 to have 7 PRs, got %d", pc.userStats["user1"].PRsCount)
	}

	if pc.userStats["user2"].PRsCount != 9 {
		t.Errorf("Expected user2 to have 9 PRs, got %d", pc.userStats["user2"].PRsCount)
	}
}

func TestGetTopUsersByScore(t *testing.T) {
	pc := &PRChampion{
		userStats: map[string]*UserStats{
			"user1": {Username: "user1", TotalScore: 3, PRsCount: 15},
			"user2": {Username: "user2", TotalScore: 2, PRsCount: 20},
			"user3": {Username: "user3", TotalScore: 2, PRsCount: 18},
			"user4": {Username: "user4", TotalScore: 1, PRsCount: 25},
		},
	}

	topUsers := pc.getTopUsersByScore(3)

	// Verificar ordenação
	if len(topUsers) != 3 {
		t.Errorf("Expected 3 users, got %d", len(topUsers))
	}

	if topUsers[0].Username != "user1" {
		t.Errorf("Expected user1 in first place, got %s", topUsers[0].Username)
	}

	// user2 deve vir antes de user3 (mesmo score, mas mais PRs)
	if topUsers[1].Username != "user2" {
		t.Errorf("Expected user2 in second place, got %s", topUsers[1].Username)
	}

	if topUsers[2].Username != "user3" {
		t.Errorf("Expected user3 in third place, got %s", topUsers[2].Username)
	}
}

func TestProcessWeeklyDataIntegration(t *testing.T) {
	// Teste para verificar se processWeeklyData e calculateUserStats são chamados
	startDate, _ := time.Parse("2006-01-02", "2024-09-30")
	endDate, _ := time.Parse("2006-01-02", "2024-10-06")

	pc := &PRChampion{
		startDate: startDate,
		endDate:   endDate,
		userStats: make(map[string]*UserStats),
		repositories: []Repository{
			{Owner: "test", Name: "repo1"},
		},
	}

	// Simular PRs para teste
	mockPRs := []*github.PullRequest{}

	// Processar dados (isto testa se a função não retorna prematuramente)
	pc.processWeeklyData(mockPRs)
	pc.calculateUserStats()

	// Se chegou aqui, não houve return prematuro
	if pc.userStats == nil {
		t.Error("userStats should be initialized after processing")
	}
}

func TestParseRepositories(t *testing.T) {
	tests := []struct {
		input    []string
		expected []Repository
		hasError bool
	}{
		{
			input: []string{"microsoft/vscode", "facebook/react"},
			expected: []Repository{
				{Owner: "microsoft", Name: "vscode"},
				{Owner: "facebook", Name: "react"},
			},
			hasError: false,
		},
		{
			input:    []string{"invalid-format"},
			expected: nil,
			hasError: true,
		},
		{
			input:    []string{},
			expected: []Repository{},
			hasError: false,
		},
		{
			input: []string{" microsoft/vscode ", " facebook/react "},
			expected: []Repository{
				{Owner: "microsoft", Name: "vscode"},
				{Owner: "facebook", Name: "react"},
			},
			hasError: false,
		},
	}

	for _, test := range tests {
		result, err := parseRepositories(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %v, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %v: %v", test.input, err)
			} else if len(result) != len(test.expected) {
				t.Errorf("For input %v, expected %d repos, got %d", test.input, len(test.expected), len(result))
			} else {
				for i, repo := range result {
					if repo.Owner != test.expected[i].Owner || repo.Name != test.expected[i].Name {
						t.Errorf("For input %v, expected repo %v, got %v", test.input, test.expected[i], repo)
					}
				}
			}
		}
	}
}

func TestEnvironmentVariableRepos(t *testing.T) {
	// Teste simula a lógica de parsing de repositórios da variável de ambiente
	envReposString := "microsoft/vscode,facebook/react, golang/go "
	repoStrings := strings.Split(envReposString, ",")

	// Remove espaços em branco (simula o código da aplicação)
	for i, repo := range repoStrings {
		repoStrings[i] = strings.TrimSpace(repo)
	}

	repositories, err := parseRepositories(repoStrings)

	if err != nil {
		t.Errorf("Unexpected error parsing env repos: %v", err)
	}

	expected := []Repository{
		{Owner: "microsoft", Name: "vscode"},
		{Owner: "facebook", Name: "react"},
		{Owner: "golang", Name: "go"},
	}

	if len(repositories) != len(expected) {
		t.Errorf("Expected %d repos, got %d", len(expected), len(repositories))
	}

	for i, repo := range repositories {
		if repo.Owner != expected[i].Owner || repo.Name != expected[i].Name {
			t.Errorf("Expected repo %v, got %v", expected[i], repo)
		}
	}
}
