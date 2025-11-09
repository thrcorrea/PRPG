// Teste manual para validar o parser de repositórios
// go run test_parser.go main.go

package main

import (
	"fmt"
	"log"
	"os"
)

func testParser() {
	fmt.Println("🧪 Testando parser de repositórios com branches que contêm barras...")

	testCases := []string{
		"microsoft/vscode:feat/rebrand-main",
		"microsoft/vscode:main,feat/rebrand-main,release/v2.0",
		"owner/repo:feature/ui/new-design,hotfix/security/patch",
		"microsoft/vscode",
	}

	for i, testCase := range testCases {
		fmt.Printf("\n%d. Testando: %s\n", i+1, testCase)

		repos, err := parseRepositories([]string{testCase})
		if err != nil {
			log.Printf("❌ Erro: %v", err)
			continue
		}

		for _, repo := range repos {
			fmt.Printf("   ✅ Owner: %s, Name: %s\n", repo.Owner, repo.Name)
			fmt.Printf("   📋 Branches: %v\n", repo.ProductionBranches)
		}
	}

	fmt.Println("\n🎉 Teste concluído!")
}

func init() {
	// Se executado diretamente, roda o teste
	if len(os.Args) > 0 && os.Args[0] == "./test_parser" {
		testParser()
		os.Exit(0)
	}
}
