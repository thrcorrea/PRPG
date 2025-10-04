# PR Champion 🏆

**PR Champion** é uma ferramenta CLI em Go que analisa PRs mergeados em repositórios GitHub e gera relatórios com rankings baseados em pontuação semanal.

## Como Funciona

- 📊 Analisa PRs mergeados em uma janela de tempo específica
- 🎯 **Suporte para múltiplos repositórios simultaneamente**
- 📅 Divide a análise por semanas (segunda a domingo)
- 🏆 O usuário com mais PRs mergeados na semana ganha **1 ponto**
- 🥇 Gera ranking final com os top 3 usuários por pontuação total agregada
- 📈 Mostra também o top 3 por número total de PRs

## Instalação

### Pré-requisitos
- Go 1.21 ou superior
- Token de acesso do GitHub

### Build da Aplicação

```bash
# Clone ou baixe o projeto
cd PR-Champion

# Instale as dependências
go mod tidy

# Compile a aplicação
go build -o pr-champion main.go
```

## Configuração

### Token do GitHub

Você precisa de um token de acesso pessoal do GitHub. Você pode:

1. **Definir como variável de ambiente** (recomendado):
   ```bash
   export GITHUB_TOKEN="seu_token_aqui"
   ```

2. **Ou passar como parâmetro** ao executar o comando:
   ```bash
   ./pr-champion --token="seu_token_aqui" [outros parâmetros]
   ```

### Configuração de Repositórios

Você pode especificar os repositórios de **3 formas diferentes**:

#### 1. Via Flag --repos (múltiplos repositórios)
```bash
./pr-champion --repos microsoft/vscode,facebook/react,golang/go --days 30
```

#### 2. Via Variável de Ambiente GITHUB_REPOS (múltiplos repositórios)
```bash
export GITHUB_REPOS="microsoft/vscode,facebook/react,golang/go"
./pr-champion --days 30
```

#### 3. Via Flags --owner e --repo (repositório único - compatibilidade)
```bash
./pr-champion --owner microsoft --repo vscode --days 30
```

### Como criar um token GitHub:
1. Vá em GitHub → Settings → Developer settings → Personal access tokens
2. Gere um novo token com as permissões:
   - `repo` (para repositórios privados) ou `public_repo` (para públicos)
   - `read:org` (se necessário para organizações)

## Uso

### Sintaxe Básica

```bash
# Repositório único
### Sintaxe Básica

```bash
# Opção 1: Via variável de ambiente (recomendado para uso frequente)
export GITHUB_REPOS="owner1/repo1,owner2/repo2"
./pr-champion [opções]

# Opção 2: Via flag
./pr-champion --repos owner1/repo1,owner2/repo2 [opções]

# Opção 3: Repositório único (compatibilidade)
./pr-champion --owner OWNER --repo REPO [opções]
```

# Múltiplos repositórios
./pr-champion --repos owner1/repo1,owner2/repo2,owner3/repo3 [opções]
```

### Parâmetros Obrigatórios

**Para repositório único:**
- `--owner, -o`: Owner/organização do repositório
- `--repo, -r`: Nome do repositório

**Para múltiplos repositórios:**
- `--repos, -R`: Lista de repositórios no formato `owner/repo` separados por vírgula

### Parâmetros Opcionais

- `--token, -t`: Token do GitHub (ou use variável `GITHUB_TOKEN`)
- `--repos, -R`: Lista de repositórios no formato owner/repo,owner2/repo2 (ou use variável `GITHUB_REPOS`)
- `--owner, -o`: Owner do repositório (para compatibilidade com repositório único)
- `--repo, -r`: Nome do repositório (para compatibilidade com repositório único)
- `--start, -s`: Data de início da análise (DD/MM/YYYY ou YYYY-MM-DD)
- `--end, -e`: Data de fim da análise (DD/MM/YYYY ou YYYY-MM-DD)
- `--days, -d`: Número de dias atrás para analisar (alternativa às datas específicas)

### Exemplos de Uso

#### Exemplo 1: Via variável de ambiente (recomendado)
```bash
export GITHUB_REPOS="microsoft/vscode,facebook/react"
./pr-champion --days 30
```

#### Exemplo 2: Múltiplos repositórios via flag
```bash
./pr-champion --repos microsoft/vscode,facebook/react,golang/go --days 14
```

#### Exemplo 3: Repositório único (compatibilidade)
```bash
./pr-champion --owner microsoft --repo vscode --days 30
```

#### Exemplo 4: Período específico com variável de ambiente
```bash
export GITHUB_REPOS="microsoft/vscode,facebook/react"
./pr-champion --start "01/09/2024" --end "30/09/2024"
```

#### Exemplo 5: Organizações relacionadas
```bash
./pr-champion --repos kubernetes/kubernetes,kubernetes/minikube,kubernetes/dashboard --days 30
```

#### Exemplo 6: Com token explícito
```bash
./pr-champion --token "ghp_xxxxxxxxxxxx" --repos google/golang,golang/go --days 60
```

## Exemplo de Saída

```
🚀 Iniciando PR Champion...
🔍 Buscando PRs mergeados de 2024-09-01 para 3 repositórios...
  📁 Analisando microsoft/vscode...
    ✅ 89 PRs encontrados em microsoft/vscode
  � Analisando facebook/react...
    ✅ 24 PRs encontrados em facebook/react
  📁 Analisando golang/go...
    ✅ 14 PRs encontrados em golang/go
�📊 Encontrados 127 PRs mergeados no período total

🏆 RELATÓRIO PR CHAMPION - 01/09/2024 a 30/09/2024
📁 Repositórios analisados (3):
   • microsoft/vscode
   • facebook/react
   • golang/go

📅 RESUMO SEMANAL:
============================================================
Semana: 02/09 - 08/09/2024
🥇 Campeão: joao_dev
   🥇 joao_dev: 12 PRs
   🥈 maria_code: 8 PRs
   🥉 pedro_git: 6 PRs

Semana: 09/09 - 15/09/2024
🥇 Campeão: maria_code
   🥇 maria_code: 15 PRs
   🥈 joao_dev: 10 PRs
   🥉 ana_prog: 7 PRs

🏅 RANKING GERAL POR PONTUAÇÃO:
============================================================
🥇 1° lugar: joao_dev
   📊 Pontuação: 3 pontos
   🏆 Vitórias semanais: 3
   📋 Total de PRs: 45

🥈 2° lugar: maria_code
   📊 Pontuação: 2 pontos
   🏆 Vitórias semanais: 2
   📋 Total de PRs: 52

🥉 3° lugar: pedro_git
   📊 Pontuação: 1 pontos
   🏆 Vitórias semanais: 1
   📋 Total de PRs: 38

📈 TOP 3 POR TOTAL DE PRS:
============================================================
🥇 1° lugar: maria_code - 52 PRs
🥈 2° lugar: joao_dev - 45 PRs
🥉 3° lugar: pedro_git - 38 PRs

✅ Relatório gerado com sucesso!
```

## Funcionalidades

### 📊 Análise Semanal
- Divide o período em semanas (segunda a domingo)
- Identifica o campeão de cada semana
- Mostra top 3 de cada semana

### 🏆 Sistema de Pontuação
- 1 ponto para quem teve mais PRs mergeados na semana
- Ranking final baseado na pontuação total
- Critério de desempate: número total de PRs

### 📈 Relatórios Múltiplos
- **Ranking por pontuação**: Top 3 usuários que mais ganharam semanas
- **Ranking por PRs**: Top 3 usuários por volume total de PRs
- **Resumo semanal**: Detalhamento semana a semana

## Limitações

- Requer token de acesso do GitHub
- Limitado pelas APIs rate limits do GitHub (5000 requests/hora para tokens autenticados)
- Analisa apenas PRs mergeados (não fechados sem merge)
- Semanas começam na segunda-feira

## Contribuição

Sinta-se à vontade para contribuir com melhorias, correções de bugs ou novas funcionalidades!

## Licença

Este projeto está sob licença MIT.
