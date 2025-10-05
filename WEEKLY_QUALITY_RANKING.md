# Sistema de Ranking Semanal por Qualidade de Comentários

## 🎯 Nova Funcionalidade Implementada

### Sistema de Pontuação Semanal por Qualidade
Adicionado um novo ranking que reconhece semanalmente o usuário com a **melhor qualidade** de comentários baseado em reações (emojis).

## 🏆 Como Funciona

### 1. Cálculo Semanal
- A cada semana, o sistema identifica quem teve a **maior pontuação ponderada** de comentários
- O vencedor da semana recebe **1 ponto** no ranking geral de qualidade

### 2. Pontuação Ponderada (Relembrar)
- **Comentário base**: 1.0 ponto
- **👍 (+1)**: +1.0 ponto adicional (total = 2.0)
- **👎 (-1)**: -2.0 pontos (pode ficar negativo, mínimo -1.0)
- **❤️🎉🚀**: +0.5 ponto adicional
- **😕👀**: -0.5 ponto

### 3. Vencedor Semanal
- Usuario com **maior soma** de pontuação ponderada na semana
- Em caso de empate, considera o número total de comentários como desempate

## 📊 Novas Estruturas de Dados

### UserStats - Campos Adicionados
```go
type UserStats struct {
    // ... campos existentes ...
    WeightedCommentWeeklyWins    int  // Vitórias semanais por qualidade
    WeightedCommentWeeklyScore   int  // Pontuação total do ranking semanal
}
```

### Exemplo de Funcionamento
```
Semana 1: João fez 5 comentários com pontuação total de 12.5 pontos
         Maria fez 3 comentários com pontuação total de 8.0 pontos
         → João ganha a semana e recebe +1 ponto no ranking

Semana 2: Maria fez 4 comentários com pontuação total de 15.0 pontos  
         João fez 6 comentários com pontuação total de 10.0 pontos
         → Maria ganha a semana e recebe +1 ponto no ranking

Ranking Final:
- João: 1 ponto (1 vitória semanal)
- Maria: 1 ponto (1 vitória semanal)
```

## 🏅 Novo Relatório

### Seção Adicionada ao Relatório
```
🏅 RANKING SEMANAL POR QUALIDADE DOS COMENTÁRIOS:
============================================================
🥇 1° lugar: user2
   🏅 Pontuação semanal: 2 pontos
   🏆 Vitórias semanais (qualidade): 2
   ⭐ Pontuação total com reações: 22.0 pontos

🥈 2° lugar: user1  
   🏅 Pontuação semanal: 1 ponto
   🏆 Vitórias semanais (qualidade): 1
   ⭐ Pontuação total com reações: 12.5 pontos
```

## 🔧 Função de Ranking

### getTopUsersByWeightedCommentWeeklyScore()
- Retorna usuários ordenados por pontuação semanal de qualidade
- Critério de desempate: número de vitórias semanais
- Filtra apenas usuários com pontuação > 0

## ✅ Benefícios

1. **Reconhecimento Semanal**: Valoriza consistência na qualidade
2. **Incentivo à Qualidade**: Encoraja comentários mais úteis e bem recebidos
3. **Competição Saudável**: Cria um ranking específico para qualidade
4. **Visibilidade**: Destaca colaboradores que fazem comentários valiosos
5. **Complementaridade**: Funciona junto com os rankings existentes

## 🎯 Diferenças dos Rankings

| Ranking                 | O que Mede                | Como Pontua                                              |
| ----------------------- | ------------------------- | -------------------------------------------------------- |
| **Comentários Simples** | Quantidade de comentários | 1 ponto por vitória semanal (quem mais comentou)         |
| **Qualidade Total**     | Pontuação total ponderada | Soma de todas as reações recebidas                       |
| **🆕 Qualidade Semanal** | Consistência na qualidade | 1 ponto por vitória semanal (melhor qualidade da semana) |

## 🧪 Testes Implementados

- ✅ Ranking ordena corretamente por pontuação semanal
- ✅ Considera apenas usuários com pontuação > 0
- ✅ Usa vitórias semanais como critério de desempate
- ✅ Integração com dados de teste existentes

## 📈 Exemplo de Uso Real

```bash
# Executar análise
./prpg --repos owner/repo1,owner/repo2 --days 30

# O relatório agora incluirá:
# 1. Rankings existentes (PRs, comentários simples, qualidade total)
# 2. 🆕 Ranking semanal por qualidade de comentários
# 3. Detalhes de vitórias semanais por qualidade
```

Essa implementação cria um incentivo adicional para que os desenvolvedores façam comentários mais úteis e bem recebidos pela equipe, promovendo uma cultura de colaboração de qualidade! 🚀
