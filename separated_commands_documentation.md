# Separação de Comandos - PR Champion

## Implementação Concluída! ✅

A aplicação PR Champion agora possui comandos separados para carregamento de dados e geração de relatórios.

## Novos Comandos

### 1. `pr-champion load`
**Carrega dados da API do GitHub e salva no banco**

```bash
# Carrega dados dos últimos 30 dias (padrão)
./pr-champion load --repos "microsoft/vscode:main|master"

# Carrega dados de um período específico
./pr-champion load --repos "microsoft/vscode:main|master" --start "01/11/2025" --end "16/11/2025"

# Carrega dados dos últimos 7 dias
./pr-champion load --repos "microsoft/vscode:main|master" --days 7
```

**Flags disponíveis:**
- `--token, -t`: Token do GitHub (ou use `GITHUB_TOKEN` env var)
- `--repos, -R`: Lista de repositórios
- `--owner, -o`: Owner do repositório (compatibilidade)
- `--repo, -r`: Nome do repositório (compatibilidade) 
- `--start, -s`: Data de início (DD/MM/YYYY ou YYYY-MM-DD)
- `--end, -e`: Data de fim (DD/MM/YYYY ou YYYY-MM-DD) - padrão: hoje
- `--days, -d`: Número de dias atrás para analisar

### 2. `pr-champion report`
**Gera relatório baseado nos dados salvos no banco**

```bash
# Gera relatório com todos os dados salvos
./pr-champion report

# Gera relatório filtrado por período
./pr-champion report --start "01/11/2025" --end "16/11/2025"

# Gera relatório dos últimos 7 dias
./pr-champion report --days 7
```

**Flags disponíveis:**
- `--start, -s`: Data de início para filtrar dados (opcional)
- `--end, -e`: Data de fim para filtrar dados (opcional)
- `--days, -d`: Número de dias atrás para filtrar dados (opcional)

### 3. `pr-champion clear`
**Limpa completamente o banco de dados**

```bash
# Remove todas as tabelas do banco
./pr-champion clear
```

## Fluxo de Trabalho

### 1. Carregamento Inicial
```bash
# Carrega dados dos últimos 30 dias
./pr-champion load --repos "microsoft/vscode:main|master,facebook/react:main"
```

### 2. Geração de Relatórios
```bash
# Gera relatório com todos os dados
./pr-champion report

# Ou filtra por período específico
./pr-champion report --days 7
```

### 3. Atualizações Incrementais
```bash
# Carrega apenas dados novos (do último carregamento até hoje)
./pr-champion load --repos "microsoft/vscode:main|master,facebook/react:main" --start "15/11/2025"
```

### 4. Limpeza (se necessário)
```bash
# Remove todos os dados e recomeça
./pr-champion clear
```

## Vantagens da Separação

✅ **Performance**: Relatórios rápidos sem chamadas à API
✅ **Flexibilidade**: Gerar múltiplos relatórios com diferentes filtros
✅ **Confiabilidade**: Dados persistidos localmente
✅ **Economia de API**: Reduz calls desnecessários ao GitHub
✅ **Análise Histórica**: Acumula dados ao longo do tempo

## Estrutura do Banco

O banco SQLite (`./data/comments.db`) armazena:
- **PRs**: Informações dos pull requests mergeados
- **Comments**: Comentários dos PRs
- **Reviews**: Reviews e aprovações
- **Reactions**: Reações aos comentários

## Migração do Comando Antigo

**Antes (comando único)**:
```bash
./pr-champion --repos "repo1,repo2" --start "01/11/2025" --end "16/11/2025"
```

**Agora (separado)**:
```bash
# 1. Carrega dados
./pr-champion load --repos "repo1,repo2" --start "01/11/2025" --end "16/11/2025"

# 2. Gera relatório
./pr-champion report
```
