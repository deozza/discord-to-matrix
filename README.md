# Discord to Matrix

- cloner le repository

```bash
git clone https://forgejo.letsjam.dev/letsjam/discord-to-matrix
cd discord-to-matrix
```
- faire un backup avec [discordchatexporter](https://github.com/tyrrrz/discordchatexporter)

```bash
docker run --rm -v ./discord:/out tyrrrz/discordchatexporter:stable exportguild -t <token_discord> -f json -g <id_guilde> --parallel <nombre_chans_a_traiter_en_meme_temps> --media true --reuse-media true --include-vc false --after <date_min_messages>
```

- configurer dans le fichier `index.php` :
  - l'adresse du serveur matrix cible
  - le login d'un compte matrix sur ce serveur
  - le mot de passe de ce compte

- lancer le projet

```bash
docker compose up
```
