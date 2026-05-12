#!/bin/bash
# Uso: ./scripts/run_quinzena.sh "01/05/2026" "15/05/2026"
# Cron (dias 1 e 16 às 09:00):
#   0 9 1,16 * * cd /path/to/pr-champion && ./scripts/run_quinzena.sh "XX/XX/XXXX" "XX/XX/XXXX" >> ./logs/quinzena.log 2>&1
set -e

START=$1
END=$2

if [ -z "$START" ] || [ -z "$END" ]; then
    echo "❌ Uso: $0 \"DD/MM/YYYY\" \"DD/MM/YYYY\""
    echo "   Exemplo: $0 \"01/05/2026\" \"15/05/2026\""
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$ROOT_DIR"

if [ ! -f "./pr-champion" ]; then
    echo "🔨 Binário não encontrado, compilando..."
    go build -o pr-champion main.go
fi

echo "🏆 Gerando quinzena: $START → $END"
./pr-champion quinzena --start "$START" --end "$END"
