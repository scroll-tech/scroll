# Coordinator

This repo contains the Scroll coordinator.

## Build

```bash
make clean
make coordinator
```

## Start

* use default ports and config.json

```bash
./build/bin/coordinator --http
```

* use specified ports and config.json

```bash
./build/bin/coordinator --config ./config.json --http --http.addr localhost --http.port 8390
```
