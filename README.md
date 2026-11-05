# QEMU Brisa — Gerenciador de Máquinas Virtuais

![Platform: macOS | Linux](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-0078D7?style=flat-square)
![macOS](https://img.shields.io/badge/macOS-Apple%20Silicon%20%7C%20Intel-000000?style=flat-square&logo=apple&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-x86__64%20%7C%20ARM64-FCC624?style=flat-square&logo=linux&logoColor=black)
[![Powered by Brisa](https://img.shields.io/badge/powered%20by-Brisa-8A2BE2?style=flat-square)](https://github.com/carlosxfelipe/brisa)
![License: MIT](https://img.shields.io/badge/license-MIT-blue?style=flat-square)

Interface desktop nativa, moderna e leve para gerenciar instâncias do **QEMU** no **macOS** e **Linux**, desenvolvida em **Go** com o [**Brisa**](https://github.com/carlosxfelipe/brisa) — micro-framework desktop zero-dependências também criado por mim.

> O [**Brisa**](https://github.com/carlosxfelipe/brisa) permite construir aplicações desktop modernas combinando Go no backend com HTML/CSS/JS no frontend, com IPC fortemente tipado (padrão REPR), streaming de eventos via SSE, temas desktop nativos e empacotamento standalone sem dependências externas.

---

## 💻 Plataformas Suportadas

O **QEMU Brisa** foi arquitetado exclusivamente para sistemas operacionais baseados em Unix (**macOS** e **Linux**), integrando-se diretamente aos aceleradores nativos de virtualização do kernel de cada plataforma:

| Sistema Operacional | Acelerador de Hardware                 | Display Gráfico                 | Arquiteturas Alvo                          |
| :------------------ | :------------------------------------- | :------------------------------ | :----------------------------------------- |
| **macOS**           | **HVF** (Apple Hypervisor.framework)   | Cocoa Nativo (`-display cocoa`) | `arm64` (Apple Silicon) e `x86_64` (Intel) |
| **Linux**           | **KVM** (Kernel-based Virtual Machine) | X11 / Wayland                   | `x86_64` e `aarch64`                       |

> [!NOTE]
> **O Windows não é suportado.** A aplicação utiliza convenções POSIX, caminhos Unix (`~/.qemu-brisa/disks`), firmware EDK2 de distribuições Unix e aceleradores específicos do macOS (`hvf`) e do Linux (`kvm`).

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

Certifique-se de ter o **QEMU** instalado no sistema:

- **No macOS** (via Homebrew):

  ```bash
  brew install qemu
  ```

- **No Linux** (Ubuntu/Debian, Fedora ou Arch):

  ```bash
  # Ubuntu / Debian
  sudo apt install qemu-system qemu-utils

  # Fedora
  sudo dnf install qemu-system-x86 qemu-system-aarch64 qemu-img

  # Arch Linux
  sudo pacman -S qemu-desktop
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

### 3. Gerar Pacote Instalável / App Nativo

- **No macOS (Gera o bundle `QEMU Brisa.app` com ícone e metadados)**:

  ```bash
  go run build_mac.go
  open "QEMU Brisa.app"
  ```

- **No Linux (Gera o executável standalone `.AppImage`)**:
  ```bash
  go run build_linux.go
  ```

### 4. Atualizar o framework Brisa

Para atualizar o Brisa diretamente para o último commit da branch `main` no GitHub:

```bash
./update-brisa.sh
```

### 5. Formatar o código

Para formatar e manter todo o código Go do projeto devidamente padronizado:

```bash
go fmt ./...
```

---

## 📄 Licença

Este projeto está sob a licença [MIT](LICENSE). Veja o arquivo [LICENSE](LICENSE) para mais detalhes.
