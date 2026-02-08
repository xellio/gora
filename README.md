# GoRa - Go RAG

**Go** based **R**etrieval **A**ugmented **G**eneration for your documentation.

> [!WARNING]  
> **NOTE:** This is a very early, unstable version. Use at your own risk.

GoRa allows you to chat with your local documentation by leveraging the power of LLMs and Vector Databases. It uses Ollama for intelligence and Redis for lightning-fast semantic search.

## Requirements
To run GoRa, you need the following components:
- **Go** (1.24.4 or higher)
- **Docker & Docker Compose** (for Redis Stack)
- **Ollama** with the following models (models can be changed in `config.yml`):
  - `nomic-embed-text`: For high-performance embeddings.
  - `gpt-oss:20b`: For generating precise, context-aware answers.
  
## Getting started

We provide a `Makefile` to simplify all common tasks.

### 1. Spin up the infrastructure  
Start the Redis Vector Database (includes Redis Insight at http://localhost:8001):
```bash
make up
```

### 2. Prepare your data  
Place your documentation (Markdown or Text files) into the /data directory. The system will automatically parse, chunk, and generate synthetic questions for these files to improve search accuracy.

### 3. Populate the vector database  
Convert your text into vectors and store them in Redis:
```
make setup
```

### 4. Start the conversation
Run the interactive CLI to chat with your documents:
```
make run
```

## Makefile Commands Overview

| Command | Description |
| --- | --- |
| `make up` | Starts the Redis Stack via docker compose |
| `make down` | Stops the Redis infrastructure. |
| `make setup` | Chunks your `/data` and populates the Redis vector store |
| `make run` | Starts the interactive GoRa CLI |
| `make build` | Compiles binaries into the `/bin` directory |
| `make status` | Shows the status of the docker containers |
| `make wipe` | Deletes all data and indexes from Redis for a fresh start |
| `make backup` | Creates a Redis snapshot and stores it in a backup/ directory |
| `make restore` | Restores a Redis snapshot, located in the backup/ directory |
| `make clean` | Removes compiled binaries and temporary files |

## Configuration
Adjust settings in `config.yml` to change models, Redis connection strings, or chunk sizes.