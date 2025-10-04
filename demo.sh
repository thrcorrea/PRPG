#!/bin/bash

# Script de demonstração do PR Champion
# Este script mostra como usar a aplicação com diferentes parâmetros

echo "🏆 PR CHAMPION - DEMONSTRAÇÃO"
echo "=============================="
echo ""

# Verifica se o GITHUB_TOKEN está configurado
if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ GITHUB_TOKEN não está configurado!"
    echo "💡 Para configurar:"
    echo "   1. Vá em https://github.com/settings/tokens"
    echo "   2. Crie um novo token com permissão 'repo' ou 'public_repo'"
    echo "   3. Execute: export GITHUB_TOKEN='seu_token_aqui'"
    echo ""
    exit 1
fi

echo "✅ GITHUB_TOKEN configurado"
echo ""

# Compila a aplicação se necessário
if [ ! -f "./pr-champion" ]; then
    echo "🔨 Compilando aplicação..."
    make build
    echo ""
fi

echo "🎯 EXEMPLO 1: Repositório único - microsoft/vscode (últimos 7 dias)"
echo "================================================================="
./pr-champion --owner microsoft --repo vscode --days 7
echo ""
echo "Pressione Enter para continuar..."
read

echo "🎯 EXEMPLO 2: Múltiplos repositórios - Ecossistema React (últimos 14 dias)"
echo "=========================================================================="
./pr-champion --repos facebook/react,vercel/next.js,remix-run/remix --days 14
echo ""
echo "Pressione Enter para continuar..."
read

echo "🎯 EXEMPLO 3: Múltiplos repositórios - Linguagens modernas (últimos 21 dias)"
echo "============================================================================"
./pr-champion --repos golang/go,rust-lang/rust,microsoft/TypeScript --days 21
echo ""
echo "Pressione Enter para continuar..."
read

echo "🎯 EXEMPLO 4: Período específico - Ecossistema Kubernetes (formato brasileiro)"
echo "=============================================================================="
echo "Analisando repositórios Kubernetes do dia 01/10/2024 até hoje"
./pr-champion --repos kubernetes/kubernetes,kubernetes/minikube --start "01/10/2024"
echo ""

echo "✅ Demonstração concluída!"
echo "📚 Para mais opções, consulte: ./pr-champion --help"
echo ""
echo "💡 Exemplos adicionais interessantes:"
echo "   # Ecossistema Microsoft:"
echo "   ./pr-champion --repos microsoft/vscode,microsoft/TypeScript,microsoft/playwright --days 14"
echo ""
echo "   # Ferramentas DevOps:"
echo "   ./pr-champion --repos docker/docker-ce,hashicorp/terraform,kubernetes/kubernetes --days 30"
