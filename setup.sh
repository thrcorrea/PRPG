#!/bin/bash

# Script de setup do PR Champion
# Este script ajuda a configurar a aplicação pela primeira vez

echo "🚀 PR CHAMPION - SETUP INICIAL"
echo "==============================="
echo ""

# Verificar se Go está instalado
if ! command -v go &> /dev/null; then
    echo "❌ Go não está instalado. Instale Go primeiro: https://golang.org/dl/"
    exit 1
fi

echo "✅ Go encontrado: $(go version)"

# Verificar se estamos no diretório correto
if [ ! -f "main.go" ]; then
    echo "❌ Execute este script no diretório do PR Champion"
    exit 1
fi

# Instalar dependências
echo ""
echo "📦 Instalando dependências..."
go mod tidy

# Criar arquivo .env se não existir
if [ ! -f ".env" ]; then
    echo ""
    echo "🔧 Criando arquivo .env..."
    cp .env.example .env
    echo "✅ Arquivo .env criado"
else
    echo ""
    echo "⚠️  Arquivo .env já existe"
fi

# Compilar aplicação
echo ""
echo "🔨 Compilando aplicação..."
make build

echo ""
echo "✅ SETUP CONCLUÍDO!"
echo ""
echo "📝 PRÓXIMOS PASSOS:"
echo "1. Edite o arquivo .env e configure seu GITHUB_TOKEN:"
echo "   - Vá em: https://github.com/settings/tokens"
echo "   - Crie um token com permissão 'repo' ou 'public_repo'"
echo "   - Edite .env e substitua 'seu_token_aqui' pelo seu token"
echo ""
echo "2. Configure os repositórios que deseja analisar em GITHUB_REPOS no arquivo .env"
echo ""
echo "3. Execute sua primeira análise:"
echo "   ./pr-champion --days 7"
echo ""
echo "💡 COMANDOS ÚTEIS:"
echo "   make run-env         # Executa com config do .env (7 dias)"
echo "   make run-env-month   # Análise mensal"
echo "   make demo            # Demonstração rápida"
echo "   make help            # Ver todos os comandos"
echo ""
echo "📚 DOCUMENTAÇÃO:"
echo "   README.md            # Documentação completa"
echo "   QUICKSTART.md        # Guia rápido"
echo "   ENV_SETUP.md         # Configurações avançadas"
