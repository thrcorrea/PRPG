# Teste de Branches com Barras

## Implementação Atualizada

✅ **Parsing melhorado**: Agora usa `strings.Index()` em vez de `strings.Split()` para lidar com barras nas branches

✅ **Suporte completo**: Branches como `feat/rebrand-main`, `release/v2.0`, `hotfix/security-patch`

## Exemplos de Branches Suportadas

### Branches Simples
- `main`
- `master`
- `production`

### Branches com Prefixos
- `feat/rebrand-main`
- `release/v2.0` 
- `hotfix/security-patch`
- `develop/new-feature`

### Branches com Múltiplos Segmentos
- `feature/ui/rebrand-main`
- `release/2024/q4`

## Exemplos de Uso

```bash
# Branch com barra única
./pr-champion --repos "microsoft/vscode:feat/rebrand-main" --start-date "01/11/2025" --end-date "09/11/2025"

# Múltiplas branches, algumas com barras
./pr-champion --repos "microsoft/vscode:main|feat/rebrand-main|release/v2.0" --start-date "01/11/2025" --end-date "09/11/2025"

# Múltiplos repositórios com branches complexas
./pr-champion --repos "microsoft/vscode:feat/rebrand-main|main,facebook/react:main|release/v18" --start-date "01/11/2025" --end-date "09/11/2025"
```

## Como Funciona o Parser

1. **Encontra o `:"`**: Usa `strings.Index()` para localizar onde terminam owner/repo
2. **Separa owner/repo**: Usa apenas a primeira barra para dividir owner e repo
3. **Processa branches**: Tudo após `:` é tratado como lista de branches separadas por vírgula
4. **Preserva barras**: Branches mantêm suas barras intactas

## Formato Detalhado

```
owner/repo:branch1,branch2,branch3
│     │   │                     │
│     │   │                     └─ Branch 3 (pode conter barras)
│     │   └─ Branch 1 (pode conter barras)  
│     └─ Nome do repositório
└─ Owner/organização
```
