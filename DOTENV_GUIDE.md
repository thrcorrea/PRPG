# Guia Completo do Arquivo .env

## 🔧 Configuração Automática

O PR Champion agora carrega automaticamente as configurações do arquivo `.env`, eliminando a necessidade de definir variáveis de ambiente manualmente.

## 🚀 Setup Rápido

### 1. Primeira vez
```bash
# Clone/baixe o projeto
cd PR-Champion

# Execute o setup automático
./setup.sh

# Ou configure manualmente:
make setup-env
```

### 2. Configure o arquivo .env
```bash
# Edite o arquivo .env
nano .env

# Configure pelo menos:
GITHUB_TOKEN=seu_token_aqui
GITHUB_REPOS=microsoft/vscode,facebook/react,golang/go
```

### 3. Execute a análise
```bash
# Simples assim! Usa configuração do .env automaticamente
./pr-champion --days 7

# Ou use os atalhos do Makefile
make run-env
make run-env-month
```

## 📝 Estrutura do Arquivo .env

```properties
# Token obrigatório
GITHUB_TOKEN=ghp_seu_token_aqui

# Lista de repositórios (obrigatório)
GITHUB_REPOS=owner1/repo1,owner2/repo2,owner3/repo3

# Configurações opcionais
DAYS_BACK=30
START_DATE=01/10/2024
END_DATE=31/10/2024
```

## 🎯 Exemplos de Configuração

### Frontend Development
```properties
GITHUB_TOKEN=ghp_seu_token_aqui
GITHUB_REPOS=facebook/react,vuejs/vue,angular/angular,sveltejs/svelte
```

### DevOps/Infrastructure
```properties
GITHUB_TOKEN=ghp_seu_token_aqui
GITHUB_REPOS=kubernetes/kubernetes,docker/docker-ce,helm/helm,istio/istio
```

### Microsoft Ecosystem
```properties
GITHUB_TOKEN=ghp_seu_token_aqui
GITHUB_REPOS=microsoft/vscode,microsoft/TypeScript,microsoft/playwright,microsoft/terminal
```

### Database Systems
```properties
GITHUB_TOKEN=ghp_seu_token_aqui
GITHUB_REPOS=postgres/postgres,mongodb/mongo,elastic/elasticsearch,redis/redis
```

## 🔄 Workspaces Múltiplos

Você pode ter diferentes configurações para diferentes projetos:

```bash
# Projeto Frontend
cp .env .env.frontend
# Edite .env.frontend com repos de frontend

# Projeto DevOps  
cp .env .env.devops
# Edite .env.devops com repos de infra

# Use conforme necessário:
cp .env.frontend .env && ./pr-champion --days 7
cp .env.devops .env && ./pr-champion --days 14
```

## 🛠️ Comandos Úteis

### Com arquivo .env configurado:
```bash
# Análise semanal (padrão)
make run-env

# Análise mensal
make run-env-month

# Demonstração rápida
make demo

# Período customizado
./pr-champion --start "01/10/2024" --end "31/10/2024"
```

### Ainda funciona com flags (override do .env):
```bash
# Sobrescreve a configuração do .env
./pr-champion --repos golang/go,rust-lang/rust --days 30

# Usa token do .env mas repos específicos
./pr-champion --owner microsoft --repo vscode --days 7
```

## 🔒 Segurança

### ⚠️ IMPORTANTE: Nunca commite tokens reais!

```bash
# Adicione ao .gitignore (já incluído)
echo ".env" >> .gitignore

# Use .env.example para exemplos
cp .env .env.example
# Remove o token real do .env.example
sed -i 's/ghp_[^=]*/seu_token_aqui/' .env.example
```

### 💡 Dicas de Segurança:
1. Use tokens com permissões mínimas necessárias
2. Para repos públicos: use `public_repo` ao invés de `repo`
3. Revogue tokens não utilizados
4. Use diferentes tokens para diferentes projetos

## 📦 Distribuição

### Criar pacote com .env.example:
```bash
make dist
```

Isso cria `./dist/` com:
- Executável compilado
- .env configurado (sem token real)
- Documentação essencial

### Para usuário final:
```bash
cd dist
# Edite .env e configure seu token
nano .env
# Execute
./pr-champion --days 7
```

## 🐛 Troubleshooting

### Arquivo .env não carregado?
```bash
# Verifique se existe
ls -la .env

# Verifique formato (sem export, sem aspas)
cat .env

# Recrie se necessário
make setup-env
```

### Token inválido?
```bash
# Teste o token manualmente
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user

# Ou teste na aplicação
./pr-champion --days 1
```

### Repos não encontrados?
```bash
# Verifique formato (sem espaços extras)
echo $GITHUB_REPOS

# Teste um repo por vez
./pr-champion --owner microsoft --repo vscode --days 1
```

## 🎯 Fluxo Recomendado

1. **Setup inicial**: `./setup.sh`
2. **Configure .env**: Edite token e repos
3. **Teste**: `make demo`
4. **Uso diário**: `make run-env` ou `./pr-champion --days X`
5. **Diferentes contextos**: Múltiplos arquivos .env
6. **Distribuição**: `make dist` para outros usuários
