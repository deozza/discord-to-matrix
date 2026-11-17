# Discord to Matrix

<!--toc:start-->
- [Discord to Matrix](#discord-to-matrix)
  - [What does it do ?](#what-does-it-do)
  - [What's the difference with a bridge ?](#whats-the-difference-with-a-bridge)
  - [Requirements](#requirements)
  - [How to run](#how-to-run)
  - [Roadmap](#roadmap)
<!--toc:end-->

## What does it do ?

If you are planning to migrate from a Discord Guild to a Matrix server, this project lets you create all the corresponding rooms with the same structure and hierarchies as the Discord channels and import all the messages.

## What's the difference with a bridge ?

Something like [a bridge](https://github.com/mautrix/discord), with the help of a bot, will create Matrix rooms according to your Discord private message and selected Guild. Then, everytime a Discord message is received, the bot will post the equivalent to the created matrix rooms. At the same time, when you post message in the created Matrix rooms, it will use your Discord account to post the same message in the equivalent Discord channel.

In other words, it helps you stay in contact with your Discord friends while not using a Discord client but your Matrix server.

What it does not is let other Matrix users join the "synchronized" Matrix rooms/Discord channel. As the owner of the rooms are the Discord bridge bot and the join policy is `invitation only`.

`discord-to-matrix`, with the help of a guild backup from [discordchatexporter](https://github.com/tyrrrz/discordchatexporter), will use your Matrix account in a selected Matrix server to create spaces and rooms to replicate the structure of said guild and import all existing text and file messages. Therefore you won't lose any history from existing guild while still being able to invite users to your newly create spaces. Thus, helping you and your friends migrating from Discord to Matrix.

## Requirements

- docker and docker-compose

## How to run

- clone the repository

```bash
git clone https://forgejo.letsjam.dev/letsjam/discord-to-matrix
cd discord-to-matrix
```
- run a discord backup with [discordchatexporter](https://github.com/tyrrrz/discordchatexporter)

```bash
docker run --rm -v ./discord:/out tyrrrz/discordchatexporter:stable exportguild -t <token_discord> -f json -g <id_guilde> --parallel <number_of_chans_to_process_in_parallels> --media true --reuse-media true --include-vc false --after <date_min_messages>
```

- copy the `.env.dist` file into a `.env` file and fill the following infos:

| Variable              | Description                                                                                | Required ?                                                 |
|-----------------------|--------------------------------------------------------------------------------------------|------------------------------------------------------------|
| DISCORD_BACKUP_DIR    | Path to the folder containing the discord backup files. By default it is `/go/src/discord` | YES                                                        |
| MATRIX_SERVER_ADDRESS | URL to the targeted Matrix server                                                          | YES                                                        |
| MATRIX_USERNAME       | Username of a valid Matrix account on the previously provided server                       | NO (if `MATRIX_TOKEN` is filled)                           |
| MATRIX_PASSWORD       | Password of a valid Matrix account on the previously provided server                       | NO (if `MATRIX_TOKEN` is filled)                           |
| MATRIX_TOKEN          | Token of a valid session on the previously provided server                                 | NO (if `MATRIX_USERNAME` and `MATRIX_PASSWORD` are filled) |

- build the env and install the dependencies

```bash
docker compose up -d --build
docker compose exec discord-to-matrix go mod tidy
```

- run the script

```bash
docker compose exec discord-to-matrix go run main.go
```

## Roadmap

1. clean the code and better architecture
2. create an entrypoint to immediately run the script at the container start
3. modify the Dockerfile to integrate the [discordchatexporter](https://github.com/tyrrrz/discordchatexporter) CLI and run the guild backup at the container build
