# Clear Database - Remoção Completa das Tabelas

## Implementação Atualizada

✅ **Função ClearDatabase**: Agora remove completamente todas as tabelas do banco
✅ **Recriação Automática**: Na próxima execução, as tabelas são criadas do zero
✅ **Limpeza Total**: Remove tabelas, índices e sequências

## Como Usar

### Comando
```bash
./pr-champion --clear-database
```

ou usando a flag curta:
```bash
./pr-champion -c
```

## Comportamento Anterior vs Atual

### ❌ Antes (Apenas DELETE)
```sql
DELETE FROM reactions;
DELETE FROM comments;  
DELETE FROM reviews;
DELETE FROM prs;
```
- **Problema**: Mantinha estrutura das tabelas
- **Resultado**: Cache zerado mas tabelas existentes

### ✅ Agora (DROP TABLE)
```sql
DROP TABLE IF EXISTS reactions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS reviews; 
DROP TABLE IF EXISTS prs;
DROP INDEX IF EXISTS [todos os índices];
```
- **Vantagem**: Remove completamente toda estrutura
- **Resultado**: Banco "virgem" para próxima execução

## Sequência de Operações

1. **Remoção das Tabelas** (ordem respeitando foreign keys):
   - `reactions` (primeiro - tem FK para comments)
   - `comments`
   - `reviews`
   - `prs`

2. **Remoção dos Índices**:
   - `idx_comments_repo_pr`
   - `idx_comments_comment_id`
   - `idx_reactions_comment_id`
   - `idx_comments_cached_at`
   - `idx_prs_repo`
   - `idx_prs_repo_pr`
   - `idx_reviews_repo_pr`
   - `idx_reviews_review_id`

3. **Limpeza das Sequências**:
   - Remove registros da `sqlite_sequence`

4. **Próxima Execução**:
   - Sistema detecta tabelas ausentes
   - Chama `createTables()` automaticamente
   - Recria toda estrutura do zero

## Vantagens

✅ **Reset Completo**: Garante estado limpo total
✅ **Sem Resíduos**: Remove qualquer inconsistência de schema
✅ **Automático**: Recriação transparente na próxima execução
✅ **Seguro**: Não afeta dados de outras aplicações

## Mensagens de Log

```
🗑️  Removendo todas as tabelas do banco de dados...
🗑️  Tabela 'reactions' removida com sucesso
🗑️  Tabela 'comments' removida com sucesso
🗑️  Tabela 'reviews' removida com sucesso  
🗑️  Tabela 'prs' removida com sucesso
✅ Banco de dados completamente limpo - tabelas serão recriadas na próxima execução
✅ Banco de dados completamente limpo! As tabelas serão recriadas na próxima execução.
```

## Casos de Uso

- **Schema Changes**: Quando há mudanças na estrutura das tabelas
- **Debugging**: Para garantir estado totalmente limpo
- **Reset Total**: Quando se quer começar do zero
- **Problemas de Corrupção**: Para resolver inconsistências no cache
