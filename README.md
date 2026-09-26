Telegram bot for viewing your server's statistics


To get started, download the project and create a `.env` file containing `TG_BOT_TOKEN=your_token`;
then, build the container and save it using these commands: 
```
docker build -t server-bot:latest
docker save -o server-bot.tar server-bot:latest
```

Upload the archive to your server and create the container using the command `docker load -i server-bot.tar`.
