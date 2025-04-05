# GitHub Telegram Bot

Telegram бот на Go для отслеживания активности GitHub аккаунта. Бот отправляет уведомления о новых репозиториях и коммитах.

## Возможности

- 🔄 Мониторинг GitHub аккаунта в реальном времени
- 🆕 Уведомления о новых репозиториях
- 📝 Уведомления о новых коммитах
- ⚙️ Настраиваемый интервал проверки

## Требования

- Go 1.18 или выше
- Токен Telegram Bot API
- Токен GitHub API
- ID чата Telegram для отправки уведомлений

## Установка

1. Клонируйте репозиторий:
```bash
git clone https://github.com/yourusername/github-tg-bot.git
cd github-tg-bot
```

2. Установите зависимости:
```bash
go mod download
```

3. Создайте файл `.env` с необходимыми переменными окружения:
```
TELEGRAM_TOKEN=your_telegram_bot_token
GITHUB_TOKEN=your_github_token
GITHUB_USERNAME=username_to_monitor
TELEGRAM_CHAT_ID=your_chat_id
CHECK_INTERVAL_MINUTES=15
```

## Запуск

```bash
go run cmd/bot/main.go
```

Или соберите и запустите бинарный файл:

```bash
go build -o github-tg-bot cmd/bot/main.go
./github-tg-bot
```

## Примеры уведомлений

### Новый репозиторий

```
🆕 Обнаружены новые репозитории:
• awesome-project - Awesome project description
  URL: https://github.com/username/awesome-project
```

### Новый коммит

```
📝 Новый коммит в репозитории awesome-project:
• Сообщение: Add new feature
• Автор: username
• Дата: 2025-04-05T11:30:00Z
• URL: https://github.com/username/awesome-project/commit/abc123
```

## Команды бота

- `/start` - Запустить бота
- `/help` - Показать справку
- `/status` - Показать статус мониторинга

## Конфигурация

Бот настраивается через переменные окружения:

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| TELEGRAM_TOKEN | Токен Telegram бота | (обязательно) |
| GITHUB_TOKEN | Токен GitHub API | (обязательно) |
| GITHUB_USERNAME | Имя пользователя GitHub для мониторинга | (обязательно) |
| TELEGRAM_CHAT_ID | ID чата Telegram для отправки уведомлений | (обязательно) |
| CHECK_INTERVAL_MINUTES | Интервал проверки в минутах | 15 |
