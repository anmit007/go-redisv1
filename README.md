# 🚀 Go-Redis

A high-performance, Redis-compatible in-memory key-value store built from scratch in Go.

## ✨ Features

| Feature | Description |
|---------|-------------|
| **Event Loop** | Single-threaded I/O multiplexing using `epoll` (Linux) and `kqueue` (macOS) |
| **RESP Protocol** | Full Redis Serialization Protocol encoder/decoder |
| **Commands** | `GET`, `SET`, `INCR`, `DEL`, `TTL`, `EXPIRE`, `PING`, `BGREWRITEAOF`, `INFO` |
| **Pipelining** | Batch multiple commands in single request |
| **LFU Eviction** | Approximated using Morris probabilistic counter (8-bit) with decay |
| **Object Encoding** | INT, EMBSTR, RAW encodings like Redis |
| **Persistence** | AOF with buffered writes and background rewrite |
| **TTL** | Lazy + active expiration with probabilistic sampling |

## 📊 Benchmarks

Tested with `redis-benchmark` on Apple M-series:

| Command | Throughput | p50 Latency | p99 Latency |
|---------|------------|-------------|-------------|
| GET | **1.5M ops/sec** | 0.45ms | 1.6ms |
| SET | **149K ops/sec** | 5.1ms | 6.6ms |
| INCR | **174K ops/sec** | 4.8ms | 6.4ms |

## 🐳 Quick Start (Docker)

```bash
# Clone and run
git clone https://github.com/anmit007/go-redis.git
cd go-redis
docker-compose up -d

# Connect with redis-cli
redis-cli -p 7379

# Run benchmarks
redis-benchmark -h 127.0.0.1 -p 7379 -t get,set -n 100000 -c 50 -q
```

## 🔧 Build from Source

```bash
go build -o go-redis .
./go-redis -host 0.0.0.0 -port 7379
```

## 🎯 Example Usage

```bash
$ redis-cli -p 7379
127.0.0.1:7379> SET user:1 "John Doe"
OK
127.0.0.1:7379> GET user:1
"John Doe"
127.0.0.1:7379> SET counter 0
OK
127.0.0.1:7379> INCR counter
"1"
127.0.0.1:7379> SET session:abc token123 EX 3600
OK
127.0.0.1:7379> TTL session:abc
(integer) 3599
127.0.0.1:7379> BGREWRITEAOF
OK
```

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Event Loop                           │
│              (epoll/kqueue syscalls)                    │
├─────────────────────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐    │
│  │ Accept  │→ │  Read   │→ │  Parse  │→ │ Execute │    │
│  │ Clients │  │ Commands│  │  RESP   │  │ Commands│    │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘    │
├─────────────────────────────────────────────────────────┤
│           ┌────────────────────────────┐                │
│           │   In-Memory Store (map)    │                │
│           │   + LFU Eviction           │                │
│           │   + TTL Expiration         │                │
│           └────────────────────────────┘                │
├─────────────────────────────────────────────────────────┤
│           ┌────────────────────────────┐                │
│           │   AOF Persistence Layer    │                │
│           │   + Buffered Writes        │                │
│           │   + Background Rewrite     │                │
│           └────────────────────────────┘                │
└─────────────────────────────────────────────────────────┘
```

## 📁 Project Structure

```
go-redis/
├── main.go                 # Entry point
├── config/                 # Configuration
├── core/
│   ├── eval.go            # Command execution
│   ├── resp.go            # RESP protocol parser
│   ├── store.go           # In-memory store
│   ├── eviction.go        # LFU eviction
│   ├── aof.go             # AOF persistence
│   ├── bgrewriteaof.go    # Background rewrite
│   └── expire.go          # TTL expiration
├── server/
│   ├── async_tcp.go       # Shared server logic
│   ├── async_tcp_linux.go # epoll implementation
│   └── async_tcp_darwin.go# kqueue implementation
├── Dockerfile
└── docker-compose.yml
```

## 🔮 Roadmap

- [ ] Approximated LRU eviction
- [ ] MULTI/EXEC transactions
- [ ] Pub/Sub
- [ ] Cluster mode

## 📄 License

MIT
