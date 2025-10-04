# 🚀 Quick Start - PR Champion

## Setup Rápido

```bash
# 1. Configure seu token GitHub
export GITHUB_TOKEN="seu_token_aqui"

# 2. Configure lista de repositórios (NOVO!)
export GITHUB_REPOS="microsoft/vscode,facebook/react"

# 3. Compile a aplicação
make build

# 4. Execute análise - vai usar os repos da variável de ambiente
./pr-champion --days 7

# Ou execute especificando repos diretamente
./pr-champion --repos microsoft/vscode,facebook/react --days 7
```

## Comandos Mais Usados

### Via variável de ambiente (mais conveniente)
```bash
# Configure uma vez
export GITHUB_REPOS="owner1/repo1,owner2/repo2,owner3/repo3"

# Use sem especificar repos toda vez
./pr-champion --days 30
./pr-champion --start "01/10/2024" --end "31/10/2024"
```

### Via flag direta
```bash
./pr-champion --repos owner1/repo1,owner2/repo2 --days 30
```

### Repositório único (compatibilidade)
```bash
./pr-champion --owner OWNER --repo REPO --days 30
```

## Repositórios Populares para Testar

### Repositório único
```bash
# Microsoft VS Code (muito ativo)
./pr-champion --owner microsoft --repo vscode --days 7

# Facebook React
./pr-champion --owner facebook --repo react --days 14

# Google Go
./pr-champion --owner golang --repo go --days 30

# Kubernetes
./pr-champion --owner kubernetes --repo kubernetes --days 7
```

### Múltiplos repositórios - Exemplos temáticos
```bash
# Ecossistema Microsoft
./pr-champion --repos microsoft/vscode,microsoft/TypeScript,microsoft/playwright --days 14

# Frontend Popular
./pr-champion --repos facebook/react,vuejs/vue,angular/angular --days 30

# Linguagens de Programação
./pr-champion --repos golang/go,rust-lang/rust,python/cpython --days 30

# Ecossistema Kubernetes
./pr-champion --repos kubernetes/kubernetes,kubernetes/minikube,kubernetes/dashboard --days 14

# Ferramentas DevOps
./pr-champion --repos docker/docker-ce,hashicorp/terraform,ansible/ansible --days 21
```

## Saída Esperada

- � **Lista de repositórios** analisados
- �📅 Resumo semanal com campeões
- 🏆 Ranking por pontuação (vitórias semanais) **agregado entre todos os repos**
- 📊 Top 3 por número total de PRs **consolidado**
- 🥇🥈🥉 Medalhas e emojis para destacar resultados

## Dicas

- Use repositórios ativos para ver resultados interessantes
- **Combine repositórios relacionados** (mesmo ecossistema/organização) para análises mais interessantes
- Períodos de 7-30 dias são ideais para análise
- Repositórios com muitos colaboradores geram rankings mais competitivos
- **Múltiplos repositórios** permitem identificar desenvolvedores que contribuem em vários projetos
