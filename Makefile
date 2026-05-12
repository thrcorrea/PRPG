# Makefile para PR Champion

# Variáveis
BINARY_NAME=pr-champion
MAIN_PATH=main.go

# Comandos padrão
.PHONY: build clean install test help run-example

# Build da aplicação
build:
	@echo "🔨 Compilando PR Champion..."
	go build -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build concluído! Execute com: ./$(BINARY_NAME)"

# Instala dependências
deps:
	@echo "📦 Instalando dependências..."
	go mod tidy
	go mod download

# Limpa arquivos de build
clean:
	@echo "🧹 Limpando arquivos de build..."
	rm -f $(BINARY_NAME)
	go clean

# Instala globalmente
install: build
	@echo "🚀 Instalando globalmente..."
	go install

# Executa testes
test:
	@echo "🧪 Executando testes..."
	go test -v ./...

# Executa usando configuração do arquivo .env
run-env:
	@echo "🎯 Executando com configuração do arquivo .env"
	@echo "📋 Usando repositórios definidos em .env"
	./$(BINARY_NAME) --days 7

# Executa análise mensal com .env
run-env-month:
	@echo "🎯 Análise mensal usando arquivo .env"
	./$(BINARY_NAME) --days 30

# Executa análise semanal com .env
run-env-week:
	@echo "🎯 Análise semanal usando arquivo .env"
	./$(BINARY_NAME) --days 7

# Setup inicial do arquivo .env
setup-env:
	@echo "🔧 Configurando arquivo .env..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ Arquivo .env criado a partir do exemplo"; \
		echo "📝 Edite o arquivo .env e configure seu GITHUB_TOKEN"; \
	else \
		echo "⚠️  Arquivo .env já existe"; \
	fi

# Executa exemplo com repositório único
run-example:
	@echo "🎯 Exemplo: Analisando repositório microsoft/vscode (últimos 30 dias)"
	@echo "⚠️  Certifique-se de definir GITHUB_TOKEN antes de executar"
	./$(BINARY_NAME) --owner microsoft --repo vscode --days 30

# Executa exemplo com múltiplos repositórios
run-example-multi:
	@echo "🎯 Exemplo: Analisando múltiplos repositórios - Ecossistema React (últimos 14 dias)"
	@echo "⚠️  Certifique-se de definir GITHUB_TOKEN antes de executar"
	./$(BINARY_NAME) --repos facebook/react,vercel/next.js,remix-run/remix --days 14

# Executa exemplo com período específico
run-example-period:
	@echo "🎯 Exemplo: Analisando múltiplos repositórios - Linguagens modernas (período específico)"
	@echo "⚠️  Certifique-se de definir GITHUB_TOKEN antes de executar"
	./$(BINARY_NAME) --repos golang/go,rust-lang/rust,microsoft/TypeScript --start "01/09/2024" --end "30/09/2024"

# Verifica se o token está configurado
check-token:
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "❌ GITHUB_TOKEN não está definido!"; \
		echo "💡 Execute: export GITHUB_TOKEN='seu_token_aqui'"; \
		exit 1; \
	else \
		echo "✅ GITHUB_TOKEN está configurado"; \
	fi

# Build e executa exemplo completo com .env
demo: build
	@echo "🎬 Executando demonstração usando arquivo .env..."
	./$(BINARY_NAME) --days 7

# Mostra ajuda
help:
	@echo "📋 Comandos disponíveis:"
	@echo ""
	@echo "  make build           - Compila a aplicação"
	@echo "  make deps            - Instala dependências"
	@echo "  make clean           - Remove arquivos de build"
	@echo "  make install         - Instala globalmente"
	@echo "  make test            - Executa testes"
	@echo "  make check-token     - Verifica se GITHUB_TOKEN está configurado"
	@echo "  make setup-env       - Cria arquivo .env a partir do exemplo"
	@echo "  make run-env         - Executa usando configuração do .env (7 dias)"
	@echo "  make run-env-week    - Executa análise semanal usando .env"
	@echo "  make run-env-month   - Executa análise mensal usando .env"
	@echo "  make demo            - Build + executa demonstração usando .env (7 dias)"
	@echo "  make dist            - Cria pacote distribuível com .env"
	@echo "  make release         - Cria builds otimizados para múltiplas plataformas"
	@echo ""
	@echo "🔧 Exemplos com flags:"
	@echo "  make run-example     - Executa exemplo (microsoft/vscode, 30 dias)"
	@echo "  make run-example-multi - Executa exemplo com múltiplos repos (React ecosystem, 14 dias)"
	@echo "  make run-example-period - Executa exemplo com período específico (múltiplos repos)"
	@echo "  make help            - Mostra esta ajuda"
	@echo ""
	@echo "📚 Para mais informações, consulte o README.md"

# Comando padrão
all: deps build

# Verifica formatação e qualidade do código
lint:
	@echo "🔍 Verificando qualidade do código..."
	go fmt ./...
	go vet ./...

# Release build (otimizado) que inclui .env
release:
	@echo "🚀 Criando builds de release..."
	@if [ ! -f .env ]; then \
		echo "❌ Arquivo .env não encontrado. Execute 'make setup-env' primeiro"; \
		exit 1; \
	fi
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "✅ Builds de release criados para múltiplas plataformas!"
	@echo "📋 Certifique-se de que o arquivo .env esteja no mesmo diretório do executável"

# Gera relatório quinzenal interativo
run-quinzena: build
	@echo "🏆 Gerando relatório quinzenal..."
	@read -p "Start (DD/MM/YYYY): " START; read -p "End (DD/MM/YYYY): " END; \
	./$(BINARY_NAME) quinzena --start "$$START" --end "$$END"

# Build distribuível com .env
dist: build
	@echo "📦 Criando pacote distribuível..."
	@mkdir -p dist
	@cp $(BINARY_NAME) dist/
	@cp .env.example dist/.env
	@cp README.md dist/
	@cp QUICKSTART.md dist/
	@echo "✅ Pacote criado em ./dist/"
	@echo "💡 Para usar: cd dist && ./$(BINARY_NAME) --days 7"
