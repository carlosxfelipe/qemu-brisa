#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if [[ "$1" == "--local" ]]; then
  echo "🌬️ Alternando para desenvolvimento local do Brisa (../brisa)..."
  go mod edit -replace github.com/carlosxfelipe/brisa=../brisa
  go mod tidy
  echo "🧪 Verificando compilação com Brisa local..."
  go vet ./...
  echo "✅ Agora usando Brisa local de: ../brisa"
  exit 0
fi

echo "🌬️ Atualizando github.com/carlosxfelipe/brisa para o commit mais recente no main..."

# Se houver diretiva de replace local ou require v0.0.0 inválido, limpa para buscar do GitHub
if grep -q "replace github.com/carlosxfelipe/brisa" go.mod 2>/dev/null; then
  echo "📦 Removendo diretiva replace para buscar do GitHub remoto..."
  go mod edit -dropreplace github.com/carlosxfelipe/brisa
fi

if grep -q "github.com/carlosxfelipe/brisa v0.0.0$" go.mod 2>/dev/null; then
  go mod edit -droprequire github.com/carlosxfelipe/brisa
fi

GOPROXY=direct go get github.com/carlosxfelipe/brisa@main
go mod tidy

echo "🧪 Executando testes e verificando compilação..."
go vet ./...

VERSION=$(grep 'github.com/carlosxfelipe/brisa' go.mod | grep -v 'replace' | awk '{print $NF}')
echo "✅ Brisa atualizado com sucesso para: ${VERSION}"
echo "💡 (Para voltar a usar o código local do Brisa, use: ./update-brisa.sh --local)"
