# Correção do Parser - Separadores Distintos

## Problema Identificado ❌

**Antes**: Vírgula (`,`) usada tanto para separar repositórios quanto branches
```bash
--repos "repo1:branch1,branch2,repo2:branch3,branch4"
         ↑              ↑       ↑              ↑
         |              |       |              └─ Branch?
         |              |       └─ Repositório?  
         |              └─ Branch?
         └─ Repositório
```

## Solução Implementada ✅

**Agora**: Pipe (`|`) para separar branches, vírgula (`,`) para separar repositórios
```bash
--repos "repo1:branch1|branch2,repo2:branch3|branch4"
         ↑              ↑       ↑              ↑
         |              |       |              └─ Branch 4
         |              |       └─ Repositório 2
         |              └─ Branch 2
         └─ Repositório 1
```

## Exemplos Corretos

### Repositório Único
```bash
--repos "microsoft/vscode:main"
```

### Repositório com Múltiplas Branches
```bash
--repos "microsoft/vscode:main|master|production"
```

### Múltiplos Repositórios
```bash
--repos "microsoft/vscode:main|master,facebook/react:main"
```

### Branches com Barras
```bash
--repos "microsoft/vscode:feat/rebrand-main|main|release/v2.0"
```

### Caso Complexo
```bash
--repos "microsoft/vscode:main|feat/rebrand-main,facebook/react:main|production,google/material:master"
```

## Separadores Definidos

| Elemento      | Separador         | Exemplo                    |
| ------------- | ----------------- | -------------------------- |
| Repositórios  | `,` (vírgula)     | `repo1,repo2,repo3`        |
| Branches      | `\|` (pipe)       | `main\|master\|production` |
| Owner/Repo    | `/` (barra)       | `microsoft/vscode`         |
| Repo/Branches | `:` (dois pontos) | `microsoft/vscode:main`    |

## Vantagens

✅ **Sem Ambiguidade**: Cada separador tem função específica
✅ **Branches com Barras**: `feat/rebrand-main` funciona perfeitamente  
✅ **Múltiplos Repositórios**: Lista clara de repositórios
✅ **Retrocompatibilidade**: Repositórios simples continuam funcionando
