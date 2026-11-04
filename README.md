# QEMU Brisa — Gerenciador de Máquinas Virtuais

Interface desktop nativa, moderna e leve para gerenciar instâncias do **QEMU** no macOS (e Linux), desenvolvida em **Go** com o framework **Brisa**.

Inspirada no modelo arquitetural de `brisa/examples/main.go`, com bindings REPR tipados, barramento de eventos SSE e armazenamento persistente seguro.

---

## 🚀 Funcionalidades

- **Seleção de Imagem ISO**: Seletor de arquivos nativo do sistema operacional (`brisa.dialog.openFile` / `osascript`) com suporte a `.iso`, `.img`, `.dmg`, `.raw` e `.qcow2`.
- **Ajuste de Memória RAM**: Slider interativo de 512 MB até o limite do host, com botões de atalho rápido (1 GB, 2 GB, 4 GB, 8 GB).
- **Controle de CPU Cores**: Slider adaptativo baseado no número de núcleos reais do processador do computador host (`runtime.NumCPU()`).
- **Suporte Multi-Arquitetura**:
  - `aarch64` (ARM64): Aceleração nativa via **Apple Silicon Hypervisor framework (HVF)** com firmware UEFI EDK2.
  - `x86_64`: Emulação com chipset moderno `q35` e gráficos VirtIO.
- **Discos Virtuais QCOW2**: Criação automática de discos rígidos com alocação dinâmica (`qemu-img create -f qcow2`) salvos em `~/.qemu-brisa/disks`.
- **Display Nativo macOS**: Inicialização com `-display cocoa`, exibindo a interface gráfica da VM integrada ao macOS.
- **Preview em Tempo Real do Comando QEMU**: Visualize e copie a linha de comando exata executada em segundo plano.
- **Gerenciador de Processos & Ciclo de Vida**: Iniciar, Parar (SIGTERM/SIGKILL), status em tempo real (Executando com PID e Uptime) e notificações no desktop.
- **Console & Logs**: Acompanhe o stdout e stderr do QEMU em tempo real via Server-Sent Events (SSE).

---

## 🛠️ Como Executar

### 1. Pré-requisitos

Certifique-se de ter o QEMU instalado via Homebrew:
```bash
brew install qemu
```

### 2. Rodar a aplicação

Para iniciar diretamente com o Go:
```bash
go run main.go
```

Ou compilar o binário standalone:
```bash
go build -o qemu-brisa .
./qemu-brisa
```

Para depuração ou testes sem abrir janela gráfica:
```bash
BRISA_HEADLESS=1 ./qemu-brisa
```
(Acesse a URL impressa no terminal ou o inspetor em `/brisa/inspector`)

---

## 📂 Estrutura do Projeto

```
qemu-brisa/
├── main.go            # Ponto de entrada, configuração do Brisa e registro de RPCs
├── vm/
│   ├── types.go       # DTOs, configurações de hardware e esquemas de validação
│   └── manager.go     # Gerenciamento de processos QEMU, detecção de binários e qemu-img
├── ui/
│   └── index.html     # Interface do usuário Libadwaita / macOS com integração Brisa SDK
├── assets/
│   └── icon.png       # Ícone do aplicativo
├── go.mod             # Módulo Go
└── README.md
```
