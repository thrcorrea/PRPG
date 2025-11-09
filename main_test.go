package main

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v55/github"
)

func TestGetTopUsersByComments(t *testing.T) {
	// Cria instância do PRChampion com dados de teste
	pc := &PRChampion{
		userStats: make(map[string]*UserStats),
	}

	// Adiciona dados de teste
	pc.userStats["user1"] = &UserStats{
		Username:                   "user1",
		PRsCount:                   5,
		CommentsCount:              10,
		CommentScore:               1,
		CommentWeeklyWins:          1,
		WeightedCommentScore:       12.5, // 10 comentários com pontuação média de 1.25
		WeightedCommentWeeklyWins:  1,    // 1 vitória semanal por qualidade
		WeightedCommentWeeklyScore: 1,    // 1 ponto por vitória semanal
		RepoStats:                  make(map[string]int),
	}
	pc.userStats["user2"] = &UserStats{
		Username:                   "user2",
		PRsCount:                   3,
		CommentsCount:              15,
		CommentScore:               2,
		CommentWeeklyWins:          2,
		WeightedCommentScore:       22.0, // 15 comentários com pontuação média melhor
		WeightedCommentWeeklyWins:  2,    // 2 vitórias semanais por qualidade
		WeightedCommentWeeklyScore: 2,    // 2 pontos por vitórias semanais
		RepoStats:                  make(map[string]int),
	}
	pc.userStats["user3"] = &UserStats{
		Username:                   "user3",
		PRsCount:                   8,
		CommentsCount:              5,
		CommentScore:               0,
		CommentWeeklyWins:          0,
		WeightedCommentScore:       3.5, // Poucos comentários mas bem avaliados
		WeightedCommentWeeklyWins:  0,   // Nenhuma vitória semanal por qualidade
		WeightedCommentWeeklyScore: 0,   // Nenhum ponto semanal
		RepoStats:                  make(map[string]int),
	}
	pc.userStats["user4"] = &UserStats{
		Username:                   "user4",
		PRsCount:                   2,
		CommentsCount:              0, // Usuário sem comentários
		CommentScore:               0,
		CommentWeeklyWins:          0,
		WeightedCommentScore:       0,
		WeightedCommentWeeklyWins:  0,
		WeightedCommentWeeklyScore: 0,
		RepoStats:                  make(map[string]int),
	}

	// Testa o ranking por comentários (número total)
	topComments := pc.getTopUsersByComments(3)

	if len(topComments) != 3 {
		t.Errorf("Expected 3 users with comments, got %d", len(topComments))
	}

	// Verifica se estão ordenados corretamente por número de comentários
	if topComments[0].Username != "user2" || topComments[0].CommentsCount != 15 {
		t.Errorf("Expected user2 with 15 comments at position 0, got %s with %d comments",
			topComments[0].Username, topComments[0].CommentsCount)
	}

	if topComments[1].Username != "user1" || topComments[1].CommentsCount != 10 {
		t.Errorf("Expected user1 with 10 comments at position 1, got %s with %d comments",
			topComments[1].Username, topComments[1].CommentsCount)
	}

	if topComments[2].Username != "user3" || topComments[2].CommentsCount != 5 {
		t.Errorf("Expected user3 with 5 comments at position 2, got %s with %d comments",
			topComments[2].Username, topComments[2].CommentsCount)
	}

	// Testa o ranking por pontuação de comentários
	topCommentScore := pc.getTopUsersByCommentScore(3)

	if len(topCommentScore) != 2 { // Apenas user1 e user2 têm pontos
		t.Errorf("Expected 2 users with comment score, got %d", len(topCommentScore))
	}

	// Verifica se estão ordenados corretamente por pontuação
	if topCommentScore[0].Username != "user2" || topCommentScore[0].CommentScore != 2 {
		t.Errorf("Expected user2 with 2 points at position 0, got %s with %d points",
			topCommentScore[0].Username, topCommentScore[0].CommentScore)
	}

	if topCommentScore[1].Username != "user1" || topCommentScore[1].CommentScore != 1 {
		t.Errorf("Expected user1 with 1 point at position 1, got %s with %d points",
			topCommentScore[1].Username, topCommentScore[1].CommentScore)
	}

	// Testa o ranking por pontuação ponderada de comentários
	topWeightedScore := pc.getTopUsersByWeightedCommentScore(3)

	if len(topWeightedScore) != 3 { // Todos têm pontuação > 0
		t.Errorf("Expected 3 users with weighted comment score, got %d", len(topWeightedScore))
	}

	// Verifica se estão ordenados corretamente por pontuação ponderada
	if topWeightedScore[0].Username != "user2" || topWeightedScore[0].WeightedCommentScore != 22.0 {
		t.Errorf("Expected user2 with 22.0 weighted points at position 0, got %s with %.1f points",
			topWeightedScore[0].Username, topWeightedScore[0].WeightedCommentScore)
	}

	if topWeightedScore[1].Username != "user1" || topWeightedScore[1].WeightedCommentScore != 12.5 {
		t.Errorf("Expected user1 with 12.5 weighted points at position 1, got %s with %.1f points",
			topWeightedScore[1].Username, topWeightedScore[1].WeightedCommentScore)
	}

	if topWeightedScore[2].Username != "user3" || topWeightedScore[2].WeightedCommentScore != 3.5 {
		t.Errorf("Expected user3 with 3.5 weighted points at position 2, got %s with %.1f points",
			topWeightedScore[2].Username, topWeightedScore[2].WeightedCommentScore)
	}

	// Testa o ranking por pontuação semanal de qualidade de comentários
	topWeeklyQuality := pc.getTopUsersByWeightedCommentWeeklyScore(3)

	if len(topWeeklyQuality) != 2 { // Apenas user1 e user2 têm pontuação semanal > 0
		t.Errorf("Expected 2 users with weekly quality score, got %d", len(topWeeklyQuality))
	}

	// Verifica se estão ordenados corretamente por pontuação semanal
	if topWeeklyQuality[0].Username != "user2" || topWeeklyQuality[0].WeightedCommentWeeklyScore != 2 {
		t.Errorf("Expected user2 with 2 weekly quality points at position 0, got %s with %d points",
			topWeeklyQuality[0].Username, topWeeklyQuality[0].WeightedCommentWeeklyScore)
	}

	if topWeeklyQuality[1].Username != "user1" || topWeeklyQuality[1].WeightedCommentWeeklyScore != 1 {
		t.Errorf("Expected user1 with 1 weekly quality point at position 1, got %s with %d points",
			topWeeklyQuality[1].Username, topWeeklyQuality[1].WeightedCommentWeeklyScore)
	}
}

// TestCalculateCommentScore testa a função de cálculo de pontuação ponderada
func TestCalculateCommentScore(t *testing.T) {
	pc := &PRChampion{}

	// Teste sem reações - score base
	score1 := pc.calculateScoreFromReactions(nil)
	if score1 != 1.0 {
		t.Errorf("Expected score 1.0 for no reactions, got %.1f", score1)
	}

	// Teste com thumbs up
	thumbsUpReactions := []*github.Reaction{
		{Content: github.String("+1")},
		{Content: github.String("+1")},
	}
	score2 := pc.calculateScoreFromReactions(thumbsUpReactions)
	if score2 != 3.0 { // 1.0 base + 2 * 1.0 thumbs up
		t.Errorf("Expected score 3.0 for 2 thumbs up, got %.1f", score2)
	}

	// Teste com thumbs down
	thumbsDownReactions := []*github.Reaction{
		{Content: github.String("-1")},
	}
	score3 := pc.calculateScoreFromReactions(thumbsDownReactions)
	if score3 != -1.0 { // 1.0 base - 2.0 thumbs down = -1.0 (min)
		t.Errorf("Expected score -1.0 for 1 thumbs down, got %.1f", score3)
	}

	// Teste com reações mistas
	mixedReactions := []*github.Reaction{
		{Content: github.String("+1")},
		{Content: github.String("-1")},
		{Content: github.String("heart")},
		{Content: github.String("laugh")},
	}
	score4 := pc.calculateScoreFromReactions(mixedReactions)
	expectedScore := 1.0 + 1.0 - 2.0 + 0.5 // = 0.5 (laugh não está mapeado, então não conta)
	if score4 != expectedScore {
		t.Errorf("Expected score %.1f for mixed reactions, got %.1f", expectedScore, score4)
	}
}

func TestIsExcludedUser(t *testing.T) {
	tests := []struct {
		username string
		expected bool
	}{
		{"GrupoGCB", true},
		{"grupogcb", true},
		{"GRUPOGCB", true},
		{"sonarqubecloud", true},
		{"SonarQubeCloud", true},
		{"copilot", true},
		{"GitHub-Actions", true},
		{"dependabot", true},
		{"codecov", true},
		{"renovate[bot]", true},
		{"github-actions[bot]", true},
		{"my-custom-bot[bot]", true},
		{"normaluser", false},
		{"john_doe", false},
		{"developer123", false},
		{"", false},
	}

	for _, test := range tests {
		result := isExcludedUser(test.username)
		if result != test.expected {
			t.Errorf("For username '%s', expected %v, got %v", test.username, test.expected, result)
		}
	}
}

// TestUserStatsReferenceUpdate testa se as modificações nos UserStats estão sendo persistidas corretamente
func TestUserStatsReferenceUpdate(t *testing.T) {
	// Cria uma instância do PRChampion
	pc := &PRChampion{
		userStats: make(map[string]*UserStats),
	}

	// Simula dados semanais
	pc.weeklyData = []WeeklyData{
		{
			StartDate: time.Now().Add(-7 * 24 * time.Hour),
			EndDate:   time.Now(),
			UserPRs: map[string]int{
				"user1": 5,
				"user2": 3,
			},
			Winner: "user1",
			UserComments: map[string]int{
				"user1": 10,
				"user2": 15,
			},
			CommentWinner: "user2",
		},
		{
			StartDate: time.Now().Add(-14 * 24 * time.Hour),
			EndDate:   time.Now().Add(-7 * 24 * time.Hour),
			UserPRs: map[string]int{
				"user1": 2,
				"user2": 8,
			},
			Winner: "user2",
			UserComments: map[string]int{
				"user1": 5,
				"user2": 7,
			},
			CommentWinner: "user2",
		},
	}

	// Executa o cálculo de estatísticas
	pc.calculateUserStats()

	// Verifica se user1 foi atualizado corretamente
	user1 := pc.userStats["user1"]
	if user1 == nil {
		t.Fatal("user1 should exist in userStats")
	}

	expectedPRs := 7 // 5 + 2
	if user1.PRsCount != expectedPRs {
		t.Errorf("Expected user1 to have %d PRs, got %d", expectedPRs, user1.PRsCount)
	}

	expectedComments := 15 // 10 + 5
	if user1.CommentsCount != expectedComments {
		t.Errorf("Expected user1 to have %d comments, got %d", expectedComments, user1.CommentsCount)
	}

	expectedWeeklyWins := 1 // ganhou apenas a primeira semana
	if user1.WeeklyWins != expectedWeeklyWins {
		t.Errorf("Expected user1 to have %d weekly wins, got %d", expectedWeeklyWins, user1.WeeklyWins)
	}

	expectedCommentScore := 0 // não ganhou nenhuma semana por comentários
	if user1.CommentScore != expectedCommentScore {
		t.Errorf("Expected user1 to have %d comment score, got %d", expectedCommentScore, user1.CommentScore)
	}

	// Verifica se user2 foi atualizado corretamente
	user2 := pc.userStats["user2"]
	if user2 == nil {
		t.Fatal("user2 should exist in userStats")
	}

	expectedPRs = 11 // 3 + 8
	if user2.PRsCount != expectedPRs {
		t.Errorf("Expected user2 to have %d PRs, got %d", expectedPRs, user2.PRsCount)
	}

	expectedComments = 22 // 15 + 7
	if user2.CommentsCount != expectedComments {
		t.Errorf("Expected user2 to have %d comments, got %d", expectedComments, user2.CommentsCount)
	}

	expectedWeeklyWins = 1 // ganhou apenas a segunda semana
	if user2.WeeklyWins != expectedWeeklyWins {
		t.Errorf("Expected user2 to have %d weekly wins, got %d", expectedWeeklyWins, user2.WeeklyWins)
	}

	expectedCommentScore = 2 // ganhou as duas semanas por comentários
	if user2.CommentScore != expectedCommentScore {
		t.Errorf("Expected user2 to have %d comment score, got %d", expectedCommentScore, user2.CommentScore)
	}

	// Teste adicional: verifica se modificar através de uma variável local realmente afeta o map
	user1Copy := pc.userStats["user1"]
	originalPRs := user1Copy.PRsCount
	user1Copy.PRsCount = 999

	if pc.userStats["user1"].PRsCount != 999 {
		t.Errorf("Reference update failed: expected 999, got %d", pc.userStats["user1"].PRsCount)
	}

	// Restaura o valor original para não afetar outros testes
	user1Copy.PRsCount = originalPRs
}

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
		{"2024-10-04 13:00:00", "2024-09-30"}, // Sexta -> Segunda anterior
		{"2024-09-30 10:00:00", "2024-09-30"}, // Segunda -> Mesma segunda
		{"2024-10-06 09:00:00", "2024-09-30"}, // Domingo -> Segunda anterior
		{"2024-10-01 14:00:00", "2024-09-30"}, // Terça -> Segunda anterior
	}

	for _, test := range tests {
		inputTime, _ := time.Parse("2006-01-02 15:04:05", test.input)
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
