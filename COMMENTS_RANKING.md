# Sistema de Pontuação por Comentários

Este documento demonstra como o sistema de pontuação por comentários funciona no PR Champion.

## Funcionalidade Implementada

O PR Champion agora inclui um **sistema de pontuação por comentários** similar ao sistema de PRs, onde:
- A cada semana, o usuário que mais comentou ganha **1 ponto**
- O ranking final é baseado na pontuação acumulada
- Em caso de empate na pontuação, o critério de desempate é o número total de comentários

### Como Funciona

1. **Busca de comentários**: Para cada PR mergeado no período, a ferramenta busca:
   - Comentários regulares dos PRs (issues comments)
   - Comentários de revisão de código (review comments)

2. **Filtro de usuários**: Automaticamente exclui comentários de:
   - Bots (usuários terminados em `[bot]`)
   - Ferramentas automatizadas: `grupogcb`, `sonarqubecloud`, `copilot`
   - Outros bots comuns: `github-actions`, `dependabot`, `codecov`, `renovate`, etc.

3. **Análise semanal**: Agrupa comentários por semana e identifica o usuário que mais comentou

4. **Pontuação**: O campeão da semana por comentários ganha 1 ponto

5. **Ranking final**: Ordena usuários por pontuação total de comentários

### Exemplo de Saída

```
💬 Buscando comentários dos PRs...
💬 Total de comentários encontrados no período: 342

� RESUMO SEMANAL:
============================================================
Semana: 02/09 - 08/09/2024
🥇 Campeão PRs: joao_dev
💬 Campeão Comentários: ana_prog
   🥇 ana_prog: 25 comentários
   🥈 joao_dev: 18 comentários
   🥉 pedro_git: 12 comentários

💬 RANKING GERAL POR PONTUAÇÃO DE COMENTÁRIOS:
============================================================
🥇 1° lugar: ana_prog
   💬 Pontuação: 2 pontos
   🏆 Vitórias semanais (comentários): 2
   📝 Total de comentários: 89

🥈 2° lugar: maria_code
   💬 Pontuação: 1 pontos
   🏆 Vitórias semanais (comentários): 1
   📝 Total de comentários: 67

💬 TOP 3 POR TOTAL DE COMENTÁRIOS:
============================================================
🥇 1° lugar: ana_prog - 89 comentários
� 2° lugar: maria_code - 67 comentários
🥉 3° lugar: joao_dev - 45 comentários
```

### Casos Especiais

- Se nenhum comentário for encontrado no período, será exibida a mensagem: "Nenhum comentário encontrado no período analisado."
- Apenas usuários que fizeram pelo menos 1 comentário aparecem no ranking
- Comentários feitos fora do período analisado são ignorados
- **Comentários de bots e ferramentas automatizadas são filtrados automaticamente**

### Usuários Filtrados (Excluídos)

A ferramenta automaticamente exclui comentários dos seguintes tipos de usuários:
- **Bots do GitHub**: Qualquer usuário terminado em `[bot]`
- **Ferramentas de análise**: `grupogcb`, `sonarqubecloud`, `sonarcloud`, `codecov`
- **Assistentes de código**: `copilot`
- **Automação**: `github-actions`, `dependabot`, `renovate`, `greenkeeper`, `snyk-bot`

> 💡 **Nota**: A comparação é feita de forma case-insensitive, então `GrupoGCB`, `grupogcb` e `GRUPOGCB` são todos filtrados.

### Tipos de Comentários Incluídos

1. **Comentários gerais do PR**: Comentários feitos na conversa geral do PR
2. **Comentários de revisão**: Comentários específicos em linhas de código durante a revisão

### Benefícios

Este sistema permite identificar:
- **Consistência na participação**: Usuários que constantemente lideram semanas em comentários
- **Engajamento em revisões**: Colaboradores mais ativos em revisões de código ao longo do tempo
- **Participação equilibrada**: Sistema de pontos evita que um único usuário com muitos comentários domine permanentemente

## Rankings Disponíveis

O sistema agora oferece **dois rankings distintos** para comentários:

### 1. Ranking por Pontuação de Comentários
- **Critério**: Pontos ganhos por vitórias semanais
- **Objetivo**: Identificar usuários **consistentemente** mais ativos em comentários
- **Empate**: Resolvido pelo número total de comentários

### 2. Ranking por Total de Comentários  
- **Critério**: Número absoluto de comentários no período
- **Objetivo**: Identificar usuários com **maior volume** de participação
- **Uso**: Complementar ao ranking por pontuação

## Teste da Funcionalidade

Testes automatizados foram criados para validar:
- **`TestGetTopUsersByComments`**: Testa ambos os rankings (por total de comentários e por pontuação)
  - Ordenação correta por número de comentários
  - Ordenação correta por pontuação de comentários
  - Exclusão de usuários sem comentários/pontos
  - Limitação aos top 3 usuários solicitados
- **`TestIsExcludedUser`**: Filtro correto de usuários excluídos (bots, ferramentas automatizadas, etc.)
