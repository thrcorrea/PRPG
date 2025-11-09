# Teste de Múltiplas Branches de Produção

## Implementação Concluída

✅ **Repository struct**: Alterado de `ProductionBranch string` para `ProductionBranches []string`

✅ **parseRepositories function**: Agora suporta formato `owner/repo:branch1,branch2,branch3`

✅ **FetchMergedPRs logic**: Atualizado para verificar PRs em qualquer das branches de produção

✅ **Documentação**: Ajuda atualizada com novos formatos suportados

## Formatos Suportados

1. **Repositório simples**: `microsoft/vscode`
   - Usa branch padrão: `main`

2. **Branch única**: `microsoft/vscode:main`
   - Usa apenas a branch especificada: `main`

3. **Múltiplas branches**: `microsoft/vscode:main|master|production`
   - Aceita PRs de qualquer uma das branches: `main`, `master`, `production`

## Exemplos de Uso

```bash
# Repositório com múltiplas branches
./pr-champion --repos "microsoft/vscode:main|master" --start-date "01/12/2024" --end-date "15/12/2024"

# Múltiplos repositórios com diferentes configurações
./pr-champion --repos "microsoft/vscode:main|master,facebook/react:main|production" --start-date "01/12/2024" --end-date "15/12/2024"

# Via variável de ambiente
export GITHUB_REPOS="microsoft/vscode:main|master|production,facebook/react:main"
./pr-champion --start-date "01/12/2024" --end-date "15/12/2024"
```

## Comportamento

- PRs são aceitos se o campo `base.ref` corresponder a **qualquer** das branches de produção configuradas
- Logs mostram quais branches são aceitas para cada repositório
- PRs que não correspondem são rejeitados com log informativo
