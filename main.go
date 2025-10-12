package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v55/github"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/thrcorrea/PRPG/internal/infrastructure"
)

// Repository representa um repositório para análise
type Repository struct {
	Owner string
	Name  string
}

// UserStats representa as estatísticas de um usuário
type UserStats struct {
	Username                   string
	PRsCount                   int
	WeeklyWins                 int
	TotalScore                 int
	RepoStats                  map[string]int // PRs por repositório
	CommentsCount              int            // Total de comentários feitos pelo usuário
	CommentWeeklyWins          int            // Vitórias semanais por comentários
	CommentScore               int            // Pontuação total por comentários
	WeightedCommentScore       float64        // Pontuação ponderada por reações (👍=+2, 👎=-1)
	WeightedCommentWeeklyWins  int            // Vitórias semanais por qualidade de comentários
	WeightedCommentWeeklyScore int            // Pontuação semanal por qualidade de comentários
}

// WeeklyData representa os dados de uma semana específica
type WeeklyData struct {
	StartDate             time.Time
	EndDate               time.Time
	UserPRs               map[string]int
	Winner                string
	RepoData              map[string]map[string]int // repo -> user -> PRs
	UserComments          map[string]int            // comentários por usuário na semana
	CommentWinner         string                    // vencedor da semana por comentários
	UserWeightedComments  map[string]float64        // pontuação ponderada por usuário na semana
	WeightedCommentWinner string                    // vencedor da semana por pontuação ponderada
}

// PRChampion é a estrutura principal da aplicação
type PRChampion struct {
	client       infrastructure.GithubAdapter
	cachedClient infrastructure.CacheableGithubAdapter // Para operações de cache
	repositories []Repository
	startDate    time.Time
	endDate      time.Time
	weeklyData   []WeeklyData
	userStats    map[string]*UserStats
}

// NewPRChampion cria uma nova instância do PR Champion
func NewPRChampion(token string, repositories []Repository, startDate, endDate time.Time) (*PRChampion, error) {
	// Cria cliente com cache em banco de dados
	cachedClient, err := infrastructure.NewCachedGithubAdapter(token, "./data/comments.db")
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente com cache: %v", err)
	}

	return &PRChampion{
		client:       cachedClient,
		cachedClient: cachedClient,
		repositories: repositories,
		startDate:    startDate,
		endDate:      endDate,
		weeklyData:   []WeeklyData{},
		userStats:    make(map[string]*UserStats),
	}, nil
}

// ClearCache limpa todo o cache do banco de dados
func (pc *PRChampion) ClearCache() error {
	if pc.cachedClient == nil {
		return fmt.Errorf("cliente com cache não está disponível")
	}
	return pc.cachedClient.ClearCache()
}

// FetchMergedPRs busca todos os PRs mergeados no período especificado para todos os repositórios
func (pc *PRChampion) FetchMergedPRs() error {
	fmt.Printf("🔍 Buscando PRs mergeados de %s para %d repositórios...\n",
		pc.startDate.Format("2006-01-02"), len(pc.repositories))

	var allPRs []*github.PullRequest

	for _, repo := range pc.repositories {
		fmt.Printf("  📁 Analisando %s/%s...\n", repo.Owner, repo.Name)

		repoPRs, err := pc.client.FetchPRsForRepo(repo.Owner, repo.Name, pc.startDate, pc.endDate)
		if err != nil {
			fmt.Printf("  ⚠️  Erro ao buscar PRs do repo %s/%s: %v\n", repo.Owner, repo.Name, err)
			continue // Continua com os outros repositórios
		}

		allPRs = append(allPRs, repoPRs...)
	}

	fmt.Printf("📊 Encontrados %d PRs mergeados no período total\n", len(allPRs))

	pc.processWeeklyData(allPRs)

	// Busca comentários para todos os PRs
	if err := pc.fetchCommentsForPRs(allPRs); err != nil {
		fmt.Printf("⚠️  Erro ao buscar comentários: %v\n", err)
	}
	pc.calculateUserStats()

	return nil
}

// fetchCommentsForPRs busca comentários de todos os PRs
func (pc *PRChampion) fetchCommentsForPRs(prs []*github.PullRequest) error {
	fmt.Printf("💬 Buscando comentários dos PRs...\n")

	ctx := context.Background()
	totalComments := 0

	// Mapas para rastrear comentários por semana
	weeklyComments := make(map[string]map[string]int)             // weekKey -> username -> count
	weeklyWeightedComments := make(map[string]map[string]float64) // weekKey -> username -> weighted score
	weekStarts := make(map[string]time.Time)

	for _, pr := range prs {
		repoOwner := pr.Base.Repo.Owner.GetLogin()
		repoName := pr.Base.Repo.GetName()
		prNumber := pr.GetNumber()
		comments, err := pc.client.ListPRComments(ctx, repoOwner, repoName, prNumber)
		if err != nil {
			fmt.Printf("  ⚠️  Erro ao buscar comentários do PR #%d em %s/%s: %v\n", prNumber, repoOwner, repoName, err)
			break
		}

		for _, comment := range comments {
			commentTime := comment.CreatedAt.Time
			// Verifica se o comentário foi feito no período analisado
			if commentTime.After(pc.startDate) && commentTime.Before(pc.endDate.Add(24*time.Hour)) {
				username := comment.User.GetLogin()

				// Filtra usuários excluídos (bots, sonarqube, etc.)
				if isExcludedUser(username) {
					continue
				}

				if username == pr.User.GetLogin() {
					fmt.Println("    ❗ Comentário do autor do PR ignorado:", username)
					continue // Pula comentários feitos pelo autor do PR
				}

				// Determina a semana do comentário
				weekStart := getWeekStart(commentTime)
				weekKey := weekStart.Format("2006-01-02")

				if weeklyComments[weekKey] == nil {
					weeklyComments[weekKey] = make(map[string]int)
					weeklyWeightedComments[weekKey] = make(map[string]float64)
					weekStarts[weekKey] = weekStart
				}

				// Calcula pontuação ponderada baseada nas reações
				commentScore := pc.calculateCommentScore(ctx, repoOwner, repoName, comment.GetID())

				weeklyComments[weekKey][username]++
				weeklyWeightedComments[weekKey][username] += commentScore
				totalComments++
			}
		}

		reviewComments, err := pc.client.ListPRReviewComments(ctx, repoOwner, repoName, prNumber)
		if err != nil {
			fmt.Printf("  ⚠️  Erro ao buscar review comments do PR #%d em %s/%s: %v\n", prNumber, repoOwner, repoName, err)
			break
		}

		for _, comment := range reviewComments {
			commentTime := comment.CreatedAt.Time
			// Verifica se o comentário foi feito no período analisado
			if commentTime.After(pc.startDate) && commentTime.Before(pc.endDate.Add(24*time.Hour)) {
				username := comment.User.GetLogin()

				// Filtra usuários excluídos (bots, sonarqube, etc.)
				if isExcludedUser(username) {
					continue
				}

				// Determina a semana do comentário
				weekStart := getWeekStart(commentTime)
				weekKey := weekStart.Format("2006-01-02")

				if weeklyComments[weekKey] == nil {
					weeklyComments[weekKey] = make(map[string]int)
					weeklyWeightedComments[weekKey] = make(map[string]float64)
					weekStarts[weekKey] = weekStart
				}

				// Calcula pontuação ponderada baseada nas reações
				commentScore := pc.calculateReviewCommentScore(ctx, repoOwner, repoName, comment.GetID())

				weeklyComments[weekKey][username]++
				weeklyWeightedComments[weekKey][username] += commentScore
				totalComments++
			}
		}

	}

	// Adiciona dados de comentários às semanas existentes
	pc.processWeeklyComments(weeklyComments, weeklyWeightedComments, weekStarts)

	fmt.Printf("💬 Total de comentários encontrados no período: %d\n", totalComments)
	return nil
}

// processWeeklyComments processa os comentários por semana e identifica vencedores
func (pc *PRChampion) processWeeklyComments(weeklyComments map[string]map[string]int, weeklyWeightedComments map[string]map[string]float64, weekStarts map[string]time.Time) {
	// Adiciona dados de comentários às semanas existentes ou cria novas semanas
	for weekKey, userComments := range weeklyComments {
		weekStart := weekStarts[weekKey]

		// Encontra o vencedor da semana por comentários (contagem simples)
		var commentWinner string
		maxComments := 0
		for user, count := range userComments {
			if count > maxComments {
				maxComments = count
				commentWinner = user
			}
		}

		// Encontra o vencedor da semana por pontuação ponderada
		var weightedCommentWinner string
		maxWeightedScore := 0.0
		userWeightedComments := weeklyWeightedComments[weekKey]
		for user, score := range userWeightedComments {
			if score > maxWeightedScore {
				maxWeightedScore = score
				weightedCommentWinner = user
			}
		}

		// Procura se já existe uma semana correspondente
		found := false
		for i := range pc.weeklyData {
			if pc.weeklyData[i].StartDate.Equal(weekStart) {
				pc.weeklyData[i].UserComments = userComments
				pc.weeklyData[i].CommentWinner = commentWinner
				pc.weeklyData[i].UserWeightedComments = userWeightedComments
				pc.weeklyData[i].WeightedCommentWinner = weightedCommentWinner
				found = true
				break
			}
		}

		// Se não encontrou, cria uma nova entrada semanal apenas para comentários
		if !found {
			weekEnd := weekStart.Add(6 * 24 * time.Hour)
			pc.weeklyData = append(pc.weeklyData, WeeklyData{
				StartDate:             weekStart,
				EndDate:               weekEnd,
				UserPRs:               make(map[string]int),
				UserComments:          userComments,
				CommentWinner:         commentWinner,
				UserWeightedComments:  userWeightedComments,
				WeightedCommentWinner: weightedCommentWinner,
			})
		}
	}

	// Reordena por data
	sort.Slice(pc.weeklyData, func(i, j int) bool {
		return pc.weeklyData[i].StartDate.Before(pc.weeklyData[j].StartDate)
	})
}

// processWeeklyData processa os PRs por semana
func (pc *PRChampion) processWeeklyData(prs []*github.PullRequest) {
	// Agrupa PRs por semana
	weeklyMap := make(map[string]map[string]int)
	weekStarts := make(map[string]time.Time)

	for _, pr := range prs {
		mergedAt := pr.MergedAt.Time
		weekStart := getWeekStart(mergedAt)
		weekKey := weekStart.Format("2006-01-02")

		if weeklyMap[weekKey] == nil {
			weeklyMap[weekKey] = make(map[string]int)
			weekStarts[weekKey] = weekStart
		}

		username := pr.User.GetLogin()
		weeklyMap[weekKey][username]++
	}

	// Converte para slice de WeeklyData
	for weekKey, userPRs := range weeklyMap {
		weekStart := weekStarts[weekKey]
		weekEnd := weekStart.Add(6 * 24 * time.Hour)

		// Encontra o vencedor da semana
		var winner string
		maxPRs := 0
		for user, count := range userPRs {
			if count > maxPRs {
				maxPRs = count
				winner = user
			}
		}

		pc.weeklyData = append(pc.weeklyData, WeeklyData{
			StartDate: weekStart,
			EndDate:   weekEnd,
			UserPRs:   userPRs,
			Winner:    winner,
		})
	}

	// Ordena por data
	sort.Slice(pc.weeklyData, func(i, j int) bool {
		return pc.weeklyData[i].StartDate.Before(pc.weeklyData[j].StartDate)
	})
}

// calculateUserStats calcula as estatísticas finais dos usuários
func (pc *PRChampion) calculateUserStats() {
	for _, week := range pc.weeklyData {
		// Processa PRs
		for username, prCount := range week.UserPRs {
			if pc.userStats[username] == nil {
				pc.userStats[username] = &UserStats{
					Username:  username,
					RepoStats: make(map[string]int),
				}
			}

			stats := pc.userStats[username]
			stats.PRsCount += prCount

			if username == week.Winner {
				stats.WeeklyWins++
				stats.TotalScore++
			}
		}

		// Processa comentários
		for username, commentCount := range week.UserComments {
			if pc.userStats[username] == nil {
				pc.userStats[username] = &UserStats{
					Username:  username,
					RepoStats: make(map[string]int),
				}
			}

			stats := pc.userStats[username]
			stats.CommentsCount += commentCount

			if username == week.CommentWinner {
				stats.CommentWeeklyWins++
				stats.CommentScore++
			}
		}

		// Processa pontuação ponderada de comentários
		for username, weightedScore := range week.UserWeightedComments {
			if pc.userStats[username] == nil {
				pc.userStats[username] = &UserStats{
					Username:  username,
					RepoStats: make(map[string]int),
				}
			}

			stats := pc.userStats[username]
			stats.WeightedCommentScore += weightedScore

			// Se for o vencedor da semana por qualidade de comentários, ganha 1 ponto
			if username == week.WeightedCommentWinner {
				stats.WeightedCommentWeeklyWins++
				stats.WeightedCommentWeeklyScore++
			}
		}
	}
}

// GenerateReport gera o relatório final
func (pc *PRChampion) GenerateReport() {
	fmt.Printf("\n🏆 RELATÓRIO PR CHAMPION - %s a %s\n",
		pc.startDate.Format("02/01/2006"), pc.endDate.Format("02/01/2006"))

	// Lista dos repositórios analisados
	fmt.Printf("📁 Repositórios analisados (%d):\n", len(pc.repositories))
	for _, repo := range pc.repositories {
		fmt.Printf("   • %s/%s\n", repo.Owner, repo.Name)
	}
	fmt.Println()

	// Relatório semanal
	fmt.Println("📅 RESUMO SEMANAL:")
	fmt.Println(strings.Repeat("=", 60))

	for _, week := range pc.weeklyData {
		fmt.Printf("Semana: %s - %s\n",
			week.StartDate.Format("02/01"), week.EndDate.Format("02/01/2006"))

		// Campeão por PRs
		if week.Winner != "" {
			fmt.Printf("🥇 Campeão PRs: %s\n", week.Winner)
			// Top 3 da semana por PRs
			weekTop := pc.getTopUsersForWeek(week.UserPRs, 3)
			for i, user := range weekTop {
				medal := []string{"🥇", "🥈", "🥉"}[i]
				fmt.Printf("   %s %s: %d PRs\n", medal, user.Username, user.PRsCount)
			}
		}

		// Campeão por comentários
		if week.CommentWinner != "" {
			fmt.Printf("💬 Campeão Comentários: %s\n", week.CommentWinner)
			// Top 3 da semana por comentários
			weekTopComments := pc.getTopUsersForWeek(week.UserComments, 3)
			for i, user := range weekTopComments {
				medal := []string{"🥇", "🥈", "🥉"}[i]
				fmt.Printf("   %s %s: %d comentários\n", medal, user.Username, user.PRsCount) // PRsCount aqui representa o número de comentários
			}
		}

		// Campeão por qualidade de comentários (pontuação ponderada)
		if week.WeightedCommentWinner != "" {
			fmt.Printf("⭐ Campeão Qualidade: %s\n", week.WeightedCommentWinner)
			// Top 3 da semana por pontuação ponderada
			weekTopWeighted := pc.getTopUsersForWeekWeighted(week.UserWeightedComments, 3)
			for i, user := range weekTopWeighted {
				medal := []string{"🥇", "🥈", "🥉"}[i]
				fmt.Printf("   %s %s: %.1f pontos\n", medal, user.Username, user.WeightedCommentScore)
			}
		}

		fmt.Println()
	}

	// Ranking geral por pontuação
	fmt.Println("🏅 RANKING GERAL POR PONTUAÇÃO:")
	fmt.Println(strings.Repeat("=", 60))

	topUsers := pc.getTopUsersByScore(3)
	for i, user := range topUsers {
		position := i + 1
		medal := ""
		switch position {
		case 1:
			medal = "🥇"
		case 2:
			medal = "🥈"
		case 3:
			medal = "🥉"
		}

		fmt.Printf("%s %d° lugar: %s\n", medal, position, user.Username)
		fmt.Printf("   📊 Pontuação: %d pontos\n", user.TotalScore)
		fmt.Printf("   🏆 Vitórias semanais: %d\n", user.WeeklyWins)
		fmt.Printf("   📋 Total de PRs: %d\n\n", user.PRsCount)
	}

	// Ranking por pontuação semanal de qualidade de comentários
	fmt.Println("🏅 RANKING SEMANAL POR QUALIDADE DOS COMENTÁRIOS:")
	fmt.Println(strings.Repeat("=", 60))

	topWeightedCommentWeeklyUsers := pc.getTopUsersByWeightedCommentWeeklyScore(3)
	if len(topWeightedCommentWeeklyUsers) == 0 {
		fmt.Println("   Nenhuma vitória semanal por qualidade de comentários foi registrada no período analisado.")
	} else {
		for i, user := range topWeightedCommentWeeklyUsers {
			position := i + 1
			medal := ""
			switch position {
			case 1:
				medal = "🥇"
			case 2:
				medal = "🥈"
			case 3:
				medal = "🥉"
			}

			fmt.Printf("%s %d° lugar: %s\n", medal, position, user.Username)
			fmt.Printf("   🏅 Pontuação semanal: %d pontos\n", user.WeightedCommentWeeklyScore)
			fmt.Printf("   🏆 Vitórias semanais (qualidade): %d\n", user.WeightedCommentWeeklyWins)
			fmt.Printf("   ⭐ Pontuação total com reações: %.1f pontos\n\n", user.WeightedCommentScore)
		}
	}

	// Top 3 por número total de PRs
	fmt.Println("📈 TOP 3 POR TOTAL DE PRS:")
	fmt.Println(strings.Repeat("=", 60))

	topByPRs2 := pc.getTopUsersByPRs(3)
	for i, user := range topByPRs2 {
		position := i + 1
		medal := []string{"🥇", "🥈", "🥉"}[i]
		fmt.Printf("%s %d° lugar: %s - %d PRs\n", medal, position, user.Username, user.PRsCount)
	}
	fmt.Println()

	// Top 3 por número total de comentários
	fmt.Println("💬 TOP 3 POR TOTAL DE COMENTÁRIOS:")
	fmt.Println(strings.Repeat("=", 60))

	topByComments := pc.getTopUsersByComments(3)
	if len(topByComments) == 0 {
		fmt.Println("   Nenhum comentário encontrado no período analisado.")
	} else {
		for i, user := range topByComments {
			position := i + 1
			medal := []string{"🥇", "🥈", "🥉"}[i]
			fmt.Printf("%s %d° lugar: %s - %d comentários\n", medal, position, user.Username, user.CommentsCount)
		}
	}
	fmt.Println()

	// Estatísticas do cache
	fmt.Println("📈 ESTATÍSTICAS DO CACHE:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("💾 Sistema de cache em banco SQLite ativo")
	fmt.Println("📋 Cache de comentários e reações: 7 dias")
	fmt.Println("🗂️  Local do banco: ./data/comments.db")
	fmt.Println("💡 Use --clear-database para limpar todo o cache")
}

// getTopUsersForWeek retorna os top usuários de uma semana específica
func (pc *PRChampion) getTopUsersForWeek(userPRs map[string]int, limit int) []UserStats {
	var users []UserStats
	for username, prCount := range userPRs {
		users = append(users, UserStats{
			Username: username,
			PRsCount: prCount,
		})
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].PRsCount > users[j].PRsCount
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersForWeekWeighted retorna os top usuários por pontuação ponderada de uma semana específica
func (pc *PRChampion) getTopUsersForWeekWeighted(userWeightedComments map[string]float64, limit int) []UserStats {
	var users []UserStats
	for username, weightedScore := range userWeightedComments {
		users = append(users, UserStats{
			Username:             username,
			WeightedCommentScore: weightedScore,
		})
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].WeightedCommentScore > users[j].WeightedCommentScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByScore retorna os top usuários por pontuação
func (pc *PRChampion) getTopUsersByScore(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		users = append(users, stats)
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].TotalScore == users[j].TotalScore {
			return users[i].PRsCount > users[j].PRsCount
		}
		return users[i].TotalScore > users[j].TotalScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByPRs retorna os top usuários por número de PRs
func (pc *PRChampion) getTopUsersByPRs(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		users = append(users, stats)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].PRsCount > users[j].PRsCount
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByComments retorna os top usuários por número de comentários
func (pc *PRChampion) getTopUsersByComments(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		if stats.CommentsCount > 0 { // Apenas usuários que fizeram comentários
			users = append(users, stats)
		}
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].CommentsCount > users[j].CommentsCount
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByCommentScore retorna os top usuários por pontuação de comentários
func (pc *PRChampion) getTopUsersByCommentScore(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		if stats.CommentScore > 0 { // Apenas usuários que ganharam pontos por comentários
			users = append(users, stats)
		}
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].CommentScore == users[j].CommentScore {
			return users[i].CommentsCount > users[j].CommentsCount
		}
		return users[i].CommentScore > users[j].CommentScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByWeightedCommentScore retorna os top usuários por pontuação ponderada de comentários
func (pc *PRChampion) getTopUsersByWeightedCommentScore(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		if stats.WeightedCommentScore > 0 { // Apenas usuários com pontuação positiva
			users = append(users, stats)
		}
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].WeightedCommentScore == users[j].WeightedCommentScore {
			return users[i].CommentsCount > users[j].CommentsCount
		}
		return users[i].WeightedCommentScore > users[j].WeightedCommentScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// getTopUsersByWeightedCommentWeeklyScore retorna os top usuários por pontuação semanal de qualidade de comentários
func (pc *PRChampion) getTopUsersByWeightedCommentWeeklyScore(limit int) []*UserStats {
	var users []*UserStats
	for _, stats := range pc.userStats {
		if stats.WeightedCommentWeeklyScore > 0 { // Apenas usuários que ganharam pontos semanais por qualidade
			users = append(users, stats)
		}
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].WeightedCommentWeeklyScore == users[j].WeightedCommentWeeklyScore {
			return users[i].WeightedCommentWeeklyWins > users[j].WeightedCommentWeeklyWins
		}
		return users[i].WeightedCommentWeeklyScore > users[j].WeightedCommentWeeklyScore
	})

	if len(users) > limit {
		users = users[:limit]
	}

	return users
}

// isExcludedUser verifica se um usuário deve ser excluído da contagem de comentários
func isExcludedUser(username string) bool {
	excludedUsers := []string{
		"grupogcb",
		"sonarqubecloud",
		"copilot",
		"github-actions",
		"dependabot",
		"codecov",
		"sonarcloud",
		"renovate",
		"greenkeeper",
		"snyk-bot",
	}

	// Converte para lowercase para comparação case-insensitive
	usernameLower := strings.ToLower(username)

	for _, excluded := range excludedUsers {
		if usernameLower == excluded || strings.Contains(usernameLower, excluded) {
			return true
		}
	}

	// Verifica se termina com [bot] (padrão do GitHub para bots)
	return strings.HasSuffix(usernameLower, "[bot]")
}

// calculateCommentScore calcula a pontuação de um comentário baseada em suas reações
func (pc *PRChampion) calculateCommentScore(ctx context.Context, repoOwner, repoName string, commentID int64) float64 {
	// Busca as reações do comentário
	reactions, err := pc.client.ListIssueCommentReactions(ctx, repoOwner, repoName, commentID)
	if err != nil {
		// Se não conseguir buscar reações, conta como 1 ponto normal
		return 1.0
	}

	return pc.calculateScoreFromReactions(reactions)
}

// calculateScoreFromReactions calcula a pontuação baseada em uma lista de reações
func (pc *PRChampion) calculateScoreFromReactions(reactions []*github.Reaction) float64 {
	score := 1.0 // Pontuação base do comentário

	for _, reaction := range reactions {
		switch reaction.GetContent() {
		case "+1": // 👍
			score += 1.0 // +1 adicional (total = 2)
		case "-1": // 👎
			score -= 2.0 // -2 para neutralizar o ponto base e ainda penalizar (-1)
		case "heart", "hooray", "rocket": // ❤️ 🎉 🚀
			score += 0.5 // Reações positivas menores
		case "confused", "eyes": // 😕 👀
			score -= 0.5 // Reações neutras/negativas menores
		}
	}

	// Garante que a pontuação mínima seja -1 (para comentários muito mal recebidos)
	if score < -1.0 {
		score = -1.0
	}

	return score
}

// calculateReviewCommentScore calcula a pontuação de um review comment baseada em suas reações
func (pc *PRChampion) calculateReviewCommentScore(ctx context.Context, repoOwner, repoName string, commentID int64) float64 {
	// Busca as reações do review comment
	reactions, err := pc.client.ListPullRequestCommentReactions(ctx, repoOwner, repoName, commentID)
	if err != nil {
		// Se não conseguir buscar reações, conta como 1 ponto normal
		return 1.0
	}

	return pc.calculateScoreFromReactions(reactions)
}

// getWeekStart retorna o início da semana (segunda-feira)
func getWeekStart(t time.Time) time.Time {
	weekday := t.Weekday()
	if weekday == 0 {
		weekday = 7 // Domingo = 7
	}
	daysBack := int(weekday) - 1
	return t.Add(-time.Duration(daysBack) * 24 * time.Hour).Truncate(24 * time.Hour)
}

// parseDate converte string de data no formato DD/MM/YYYY para time.Time
func parseDate(dateStr string) (time.Time, error) {
	layouts := []string{
		"02/01/2006",
		"2006-01-02",
		"02-01-2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("formato de data inválido: %s (use DD/MM/YYYY ou YYYY-MM-DD)", dateStr)
}

// parseRepositories converte strings de repositórios para slice de Repository
func parseRepositories(repoStrings []string) ([]Repository, error) {
	var repositories []Repository

	for _, repoStr := range repoStrings {
		parts := strings.Split(repoStr, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("formato de repositório inválido: %s (use owner/repo)", repoStr)
		}

		repositories = append(repositories, Repository{
			Owner: strings.TrimSpace(parts[0]),
			Name:  strings.TrimSpace(parts[1]),
		})
	}

	return repositories, nil
}

var rootCmd = &cobra.Command{
	Use:   "pr-champion",
	Short: "PR Champion - Contabiliza PRs mergeados e gera ranking",
	Long: `PR Champion é uma ferramenta CLI que analisa PRs mergeados em repositórios GitHub
e gera relatórios com rankings baseados em pontuação semanal.

Suporta análise de repositório único ou múltiplos repositórios simultaneamente.
Cada semana, o usuário que mais teve PRs mergeados ganha 1 ponto.
O ranking final mostra os top 3 usuários por pontuação total agregada.

Repositórios podem ser especificados via:
  • Flag --repos: --repos microsoft/vscode,facebook/react
  • Variável de ambiente: GITHUB_REPOS=microsoft/vscode,facebook/react
  • Flags individuais: --owner microsoft --repo vscode`,
	Run: func(cmd *cobra.Command, args []string) {
		// Carrega variáveis do arquivo .env se existir
		if err := godotenv.Load(); err != nil {
			// Não é um erro fatal se o arquivo .env não existir
			if !os.IsNotExist(err) {
				fmt.Printf("⚠️  Aviso: Erro ao carregar .env: %v\n", err)
			}
		} else {
			fmt.Println("✅ Arquivo .env carregado com sucesso")
		}

		token, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")
		repo, _ := cmd.Flags().GetString("repo")
		reposList, _ := cmd.Flags().GetStringSlice("repos")
		startDateStr, _ := cmd.Flags().GetString("start")
		endDateStr, _ := cmd.Flags().GetString("end")
		daysBack, _ := cmd.Flags().GetInt("days")
		clearDatabase, _ := cmd.Flags().GetBool("clear-database")

		// Validação do token
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
			if token == "" {
				log.Fatal("❌ Token do GitHub é obrigatório. Use --token ou defina GITHUB_TOKEN")
			}
		}

		// Construir lista de repositórios
		var repositories []Repository
		var err error

		if len(reposList) > 0 {
			// Usar lista de repositórios da flag --repos
			repositories, err = parseRepositories(reposList)
			if err != nil {
				log.Fatalf("❌ Erro ao parsear repositórios da flag: %v", err)
			}
		} else if owner != "" && repo != "" {
			// Usar repositório único (compatibilidade)
			repositories = []Repository{{Owner: owner, Name: repo}}
		} else {
			// Tentar ler da variável de ambiente GITHUB_REPOS
			envRepos := os.Getenv("GITHUB_REPOS")
			if envRepos != "" {
				repoStrings := strings.Split(envRepos, ",")
				// Remove espaços em branco
				for i, repo := range repoStrings {
					repoStrings[i] = strings.TrimSpace(repo)
				}
				repositories, err = parseRepositories(repoStrings)
				if err != nil {
					log.Fatalf("❌ Erro ao parsear repositórios da variável GITHUB_REPOS: %v", err)
				}
				fmt.Printf("📋 Usando repositórios da variável GITHUB_REPOS: %s\n", envRepos)
			} else {
				log.Fatal("❌ Especifique repositórios usando:\n" +
					"   • --repos owner1/repo1,owner2/repo2\n" +
					"   • --owner e --repo (repositório único)\n" +
					"   • Variável GITHUB_REPOS=owner1/repo1,owner2/repo2")
			}
		}

		var startDate, endDate time.Time

		// Se foi especificado --days, calcula as datas automaticamente
		if daysBack > 0 {
			endDate = time.Now()
			startDate = endDate.Add(-time.Duration(daysBack) * 24 * time.Hour)
		} else {
			// Parse das datas
			if startDateStr == "" {
				startDate = time.Now().Add(-30 * 24 * time.Hour) // 30 dias atrás por padrão
			} else {
				startDate, err = parseDate(startDateStr)
				if err != nil {
					log.Fatalf("❌ Erro na data de início: %v", err)
				}
			}

			if endDateStr == "" {
				endDate = time.Now()
			} else {
				endDate, err = parseDate(endDateStr)
				if err != nil {
					log.Fatalf("❌ Erro na data de fim: %v", err)
				}
			}
		}

		// Validação das datas
		if endDate.Before(startDate) {
			log.Fatal("❌ Data de fim deve ser posterior à data de início")
		}

		fmt.Println("🚀 Iniciando PR Champion...")

		prChampion, err := NewPRChampion(token, repositories, startDate, endDate)
		if err != nil {
			log.Fatalf("❌ Erro ao inicializar PR Champion: %v", err)
		}

		// Garante que a conexão seja fechada no final
		defer func() {
			if prChampion.cachedClient != nil {
				prChampion.cachedClient.Close()
			}
		}()

		// Se a flag clear-database foi especificada, limpa o cache primeiro
		if clearDatabase {
			fmt.Println("🗑️  Limpando cache do banco de dados...")
			if err := prChampion.ClearCache(); err != nil {
				log.Fatalf("❌ Erro ao limpar cache: %v", err)
			}
			fmt.Println("✅ Cache limpo com sucesso!")
		}

		if err := prChampion.FetchMergedPRs(); err != nil {
			log.Fatalf("❌ Erro ao buscar PRs: %v", err)
		}

		prChampion.GenerateReport()

		fmt.Println("\n✅ Relatório gerado com sucesso!")
	},
}

func init() {
	rootCmd.Flags().StringP("token", "t", "", "Token de acesso do GitHub (ou use GITHUB_TOKEN env var)")
	rootCmd.Flags().StringP("owner", "o", "", "Owner do repositório (compatibilidade com repo único)")
	rootCmd.Flags().StringP("repo", "r", "", "Nome do repositório (compatibilidade com repo único)")
	rootCmd.Flags().StringSliceP("repos", "R", []string{}, "Lista de repositórios no formato owner/repo (ou use GITHUB_REPOS env var)")
	rootCmd.Flags().StringP("start", "s", "", "Data de início (DD/MM/YYYY ou YYYY-MM-DD)")
	rootCmd.Flags().StringP("end", "e", "", "Data de fim (DD/MM/YYYY ou YYYY-MM-DD)")
	rootCmd.Flags().IntP("days", "d", 0, "Número de dias atrás para analisar (alternativa às datas específicas)")
	rootCmd.Flags().BoolP("clear-database", "c", false, "Limpa todo o cache do banco de dados antes de executar")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
