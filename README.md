# GoRa - Go RAG

**Go** based **R**etrieval **A**ugmented **G**eneration for your documentation.

> [!WARNING]  
> **NOTE:** This is a very early, unstable version. Use at your own risk.

GoRa allows you to chat with your local documentation by leveraging the power of LLMs and Vector Databases. It uses Ollama for intelligence and Redis for lightning-fast semantic search.

## Requirements
Tp run GoRa, you need the following components:
- Go (1.24.4 or higher)
- Redis Stack (with RediSearch module, provided via `docker-compose.yml`)
- Ollama with the following models:
  - `nomic-embed-text`: For high-performance embeddings.
  - `gpt-oss:20b`: For generating precise, context-aware answers.
  
## Getting started
1. Spin up the infrastructure  
Start the Redis Vector Database (includes Redis Insight at http://localhost:8001):
```
docker-compose up -d
```
2. Prepare your data  
Place your documentation (Markdown or Text files) into the `/data` directory. The system will automatically parse and chunk these files.

3. Populate the vector database  
Convert your text into vectors and store them in Redis:
```
go run cmd/database/setup.go
```
4. Start the conversation  
Run the interactive CLI to chat with your documents:
```
go run cmd/prompt/main.go
```
