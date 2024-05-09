FROM ghcr.io/foundry-rs/foundry:latest

WORKDIR /app

COPY genesis.json /app/genesis.json

ENV ANVIL_IP_ADDR=0.0.0.0

ENTRYPOINT ["anvil", "--init", "/app/genesis.json"]
