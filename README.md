# WhatsApp gRPC Simulation 🚀

Este projeto é uma simulação de um serviço de chat (estilo WhatsApp) desenvolvido para a disciplina de **Sistemas Distribuídos**. O sistema utiliza **gRPC** para comunicação eficiente entre cliente e servidor, permitindo troca de mensagens em tempo real através de streams.

## 🛠 Tecnologias Utilizadas

- **Go (Golang):** Linguagem principal do projeto.
- **gRPC & Protocol Buffers:** Para definição de serviços e comunicação de alto desempenho.
- **PostgreSQL:** Banco de dados relacional para persistência de usuários e histórico de mensagens.
- **Redis:** Utilizado para gerenciamento de sessões e status online dos usuários em tempo real.
- **Docker & Docker Compose:** Para orquestração da infraestrutura (Banco de Dados e Cache).

## 🏗 Arquitetura do Projeto

O projeto segue uma arquitetura em camadas, facilitando a manutenção e escalabilidade:

- **Client-Server:** Comunicação baseada em chamadas remotas de procedimento (RPC) e Streams bidirecionais para notificações em tempo real.
- **Persistência:** Camada de repositório isolada para interação com o PostgreSQL.
- **Cache:** Camada dedicada ao Redis para verificação rápida de status de conexão.
- **Domain:** Definição das entidades de negócio e tipos compartilhados.

### Divisão de Pastas

```text
├── client/          # Implementação do cliente CLI interativo
├── server/          # Ponto de entrada do servidor gRPC
├── src/
│   ├── app/         # Lógica de negócio e implementação do serviço gRPC
│   ├── cache/       # Cliente e lógica do Redis
│   ├── domain/      # Entidades e regras de domínio
│   ├── pb/          # Código gerado pelo compilador protoc
│   ├── proto/       # Definições do serviço (.proto)
│   └── storage/     # Configuração do banco de dados e repositórios (SQL)
├── docker-compose.yml # Configuração do Postgres e Redis
└── go.mod           # Gerenciamento de dependências
```

## 🚀 Como Executar

Siga os passos abaixo para rodar o projeto em sua máquina local:

### 1. Subir a Infraestrutura (Postgres & Redis)

Certifique-se de ter o Docker instalado e execute:
```bash
docker-compose up -d
```

### 2. Iniciar o Servidor

Em um terminal, execute o comando para iniciar o serviço gRPC:
```bash
go run server/main.go
```
O servidor estará rodando e aguardando conexões na porta `:9090`.

### 3. Iniciar o Cliente (Múltiplas instâncias)

Abra **novos terminais** para cada usuário que deseja simular. Em cada terminal, execute:
```bash
go run client/main.go
```

### 💬 Funcionalidades no Cliente
- **Registro/Login:** Informe seu telefone para entrar ou criar um perfil.
- **Chat em Tempo Real:** Envie mensagens instantâneas para outros usuários via ID.
- **Histórico:** Visualize mensagens anteriores ao abrir uma conversa.
- **Status de Mensagem:** Notificações visuais de mensagens lidas (`[✔✔]`).
- **Comandos Úteis:**
  - `/r`: Responder à última pessoa que te mandou mensagem.
  - `/s`: Sair da conversa atual ou do aplicativo.


