# 🚀 INÍCIO RÁPIDO - PR Champion com .env

## ⚡ Setup em 3 Passos

### 1. Configuração Inicial
```bash
# Execute o setup automático
./setup.sh

# OU configure manualmente:
make setup-env
make build
```

### 2. Configure seu Token
```bash
# Edite o arquivo .env
nano .env

# Substitua a linha:
GITHUB_TOKEN=seu_token_aqui
# Por:
GITHUB_TOKEN=ghp_seu_token_real_aqui
```

**💡 Como obter um token:**
- Vá em: https://github.com/settings/tokens
- Clique em "Generate new token (classic)"
- Dê permissão `public_repo` (repos públicos) ou `repo` (todos os repos)
- Copie o token gerado

### 3. Execute!
```bash
# Usa configuração do .env automaticamente
./pr-champion --days 7

# Ou use atalhos do Makefile
make run-env        # 7 dias
make run-env-month  # 30 dias
```

## 🎯 Configurações Prontas

O arquivo `.env` já vem com configurações de exemplo. Escolha uma:

### Padrão (já configurado)
```properties
GITHUB_REPOS=microsoft/vscode,facebook/react,golang/go
```

### Frontend Frameworks
```properties
GITHUB_REPOS=facebook/react,vuejs/vue,angular/angular
```

### DevOps/Cloud Native
```properties
GITHUB_REPOS=kubernetes/kubernetes,docker/docker-ce,helm/helm
```

### Microsoft Ecosystem
```properties
GITHUB_REPOS=microsoft/vscode,microsoft/TypeScript,microsoft/playwright
```

## 📋 Comandos Essenciais

```bash
# Configuração
make setup-env      # Cria .env inicial
make build          # Compila aplicação

# Execução com .env
make run-env        # Análise 7 dias
make run-env-week   # Análise semanal
make run-env-month  # Análise mensal
make demo           # Demonstração rápida

# Utilitários
make help           # Ver todos comandos
make dist           # Criar pacote distribuível
```

## 🔧 Personalização Rápida

### Mudar repositórios:
```bash
# Edite .env e mude a linha GITHUB_REPOS
nano .env
```

### Análise de período específico:
```bash
./pr-champion --start "01/09/2024" --end "30/09/2024"
```

### Override temporário:
```bash
# Usa token do .env mas repos específicos
./pr-champion --repos torvalds/linux,microsoft/WSL --days 14
```

## 📚 Documentação Completa

- `README.md` - Documentação completa
- `QUICKSTART.md` - Guia rápido
- `DOTENV_GUIDE.md` - Guia detalhado do .env
- `ENV_SETUP.md` - Configurações avançadas
- `EXAMPLES.md` - Casos de uso práticos

## 🎉 Pronto!

Agora você tem uma aplicação totalmente configurada que:
- ✅ Carrega configurações automaticamente
- ✅ Não precisa digitar repositórios toda vez
- ✅ Mantém suas preferências salvas
- ✅ Funciona com comandos simples
