# Configurações de Ambiente - PR Champion

Este arquivo mostra como configurar diferentes cenários usando variáveis de ambiente.

## 🔧 Setup Básico

```bash
# Token obrigatório
export GITHUB_TOKEN="ghp_seu_token_aqui"

# Lista de repositórios (uma das opções abaixo)
export GITHUB_REPOS="microsoft/vscode,facebook/react,golang/go"
```

## 🎯 Cenários Predefinidos

### Frontend Frameworks
```bash
export GITHUB_REPOS="facebook/react,vuejs/vue,angular/angular"
./pr-champion --days 30
```

### Cloud Native / DevOps
```bash
export GITHUB_REPOS="kubernetes/kubernetes,docker/docker-ce,helm/helm,istio/istio"
./pr-champion --days 21
```

### Microsoft Ecosystem
```bash
export GITHUB_REPOS="microsoft/vscode,microsoft/TypeScript,microsoft/playwright,microsoft/terminal"
./pr-champion --days 14
```

### Programming Languages
```bash
export GITHUB_REPOS="golang/go,rust-lang/rust,python/cpython"
./pr-champion --days 60
```

### Database Systems
```bash
export GITHUB_REPOS="postgres/postgres,mongodb/mongo,elastic/elasticsearch"
./pr-champion --days 30
```

### Monitoring & Observability
```bash
export GITHUB_REPOS="prometheus/prometheus,grafana/grafana,jaegertracing/jaeger"
./pr-champion --days 30
```

### HashiCorp Tools
```bash
export GITHUB_REPOS="hashicorp/terraform,hashicorp/vault,hashicorp/consul"
./pr-champion --days 30
```

### React Ecosystem
```bash
export GITHUB_REPOS="facebook/react,vercel/next.js,remix-run/remix,gatsbyjs/gatsby"
./pr-champion --days 21
```

## 📁 Arquivo .bashrc/.zshrc

Para uso permanente, adicione ao seu arquivo de configuração do shell:

```bash
# PR Champion Configuration
export GITHUB_TOKEN="ghp_seu_token_aqui"

# Default repositories (altere conforme necessário)
export GITHUB_REPOS="microsoft/vscode,facebook/react,golang/go"

# Aliases úteis
alias prc="pr-champion"
alias prc-week="pr-champion --days 7"
alias prc-month="pr-champion --days 30"
alias prc-quarter="pr-champion --days 90"
```

## 🔄 Mudança Rápida de Contexto

```bash
# Análise de ecossistema frontend
export GITHUB_REPOS="facebook/react,vuejs/vue,angular/angular"
pr-champion --days 14

# Mudança para ferramentas DevOps
export GITHUB_REPOS="kubernetes/kubernetes,docker/docker-ce,terraform/terraform"
pr-champion --days 21

# Volta para configuração padrão
export GITHUB_REPOS="microsoft/vscode,facebook/react,golang/go"
pr-champion --days 30
```

## 🎮 Scripts Predefinidos

Crie scripts para cenários específicos:

### script-frontend.sh
```bash
#!/bin/bash
export GITHUB_REPOS="facebook/react,vuejs/vue,angular/angular,sveltejs/svelte"
./pr-champion --days 30
```

### script-devops.sh
```bash
#!/bin/bash
export GITHUB_REPOS="kubernetes/kubernetes,docker/docker-ce,helm/helm"
./pr-champion --days 21
```

### script-languages.sh
```bash
#!/bin/bash
export GITHUB_REPOS="golang/go,rust-lang/rust,python/cpython,microsoft/TypeScript"
./pr-champion --days 60
```

## 💡 Dicas

1. **Organize por temas**: Agrupe repositórios relacionados
2. **Use aliases**: Crie comandos curtos para uso frequente
3. **Scripts específicos**: Crie scripts para diferentes análises
4. **Combine com flags**: `GITHUB_REPOS` + `--start` e `--end` para períodos específicos
5. **Teste diferentes períodos**: Varie `--days` conforme a atividade do repositório
