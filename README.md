# Discord to Matrix

- clone the repository

```bash
git clone https://forgejo.letsjam.dev/letsjam/discord-to-matrix
cd discord-to-matrix
```
- run a discord backup with [discordchatexporter](https://github.com/tyrrrz/discordchatexporter)

```bash
docker run --rm -v ./discord:/out tyrrrz/discordchatexporter:stable exportguild -t <token_discord> -f json -g <id_guilde> --parallel <number_of_chans_to_process_in_parallels> --media true --reuse-media true --include-vc false --after <date_min_messages>
```

- build the env and install the dependencies

```bash
docker compose up -d --build
docker compose exec discord-to-matrix go mod tidy
```

- run the script

```bash
docker compose exec discord-to-matrix go run main.go
```
