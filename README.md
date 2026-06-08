# Publika Auction Bot

Telegram бот для управления аукционом. Позволяет проводить онлайн-аукционы через Telegram с системой ставок.

## 📋 Функциональность

- **Управление лотами** - просмотр и управление лотами аукциона
- **Система ставок** - размещение и управление ставками через Telegram
- **Чаты участников** - прямое общение с участниками
- **Аналитика** - просмотр зарегистрированных участников и статистики ставок
- **Мониторинг** - Prometheus метрики для отслеживания состояния бота
- **Логирование** - структурированное логирование через Graylog

## 🛠️ Технологический стек

- **Go 1.19+** - язык программирования
- **Telegram Bot API** - интеграция с Telegram
- **Redis** - кэширование и хранение состояния
- **MongoDB** - база данных (опционально)
- **Prometheus** - метрики
- **Graylog** - централизованное логирование

## 📦 Зависимости

```
github.com/go-telegram-bot-api/telegram-bot-api/v5 - Telegram Bot API
github.com/go-redis/redis/v8                         - Redis клиент
github.com/prometheus/client_golang                  - Prometheus метрики
github.com/rs/zerolog                                - Логирование
github.com/joho/godotenv                             - Загрузка .env файлов
github.com/google/uuid                               - UUID генератор
```

## 🚀 Начало работы

### Требования

- Go 1.19 или выше
- Redis
- Telegram Bot Token

### Установка

1. Клонируйте репозиторий:
```bash
git clone https://github.com/ilyakasharokov/publika-auction.git
cd publika-auction
```

2. Установите з��висимости:
```bash
go mod download
go mod tidy
```

3. Создайте файл `.env` (скопируйте из `.env.example`):
```bash
cp .env.example .env
```

4. Отредактируйте `.env` с вашими параметрами:
```env
PUBLIKA_AUCTION_BOT_TOKEN=your_bot_token_here
PUBLIKA_AUCTION_BOT_ADDR=:8002
PUBLIKA_AUCTION_BOT_UPDATE_DATA_PERIOD=5m
PUBLIKA_AUCTION_BOT_TG_ENDPOINT=https://api.telegram.org/bot%s/%s
```

5. Запустите бота:
```bash
go run ./cmd/auction
```

## 📁 Структура проекта

```
publika-auction/
├── cmd/
│   └── auction/
│       └── main.go              # Точка входа приложения
├── internal/
│   ├── bot/                     # Логика Telegram бота
│   ├── models/                  # Структуры данных
│   ├── handlers/                # HTTP обработчики
│   ├── storage/                 # Работа с БД и Redis
│   └── services/                # Бизнес-логика
├── web/
│   ├── templates/               # HTML шаблоны
│   └── static/                  # Статические ресурсы
├── migrations/                  # Миграции БД
├── go.mod                        # Go модули
├── go.sum                        # Зависимости
├── .env.example                 # Пример конфигурации
├── .gitignore                   # Git исключения
└── README.md                     # Данный файл
```

## 🔧 API эндпоинты

| Метод | Эндпоинт | Описание |
|-------|----------|----------|
| GET | `/main` | Главная страница с лотами |
| GET | `/lot/:id` | Детали лота и история ставок |
| GET | `/chats` | Список чатов с участниками |
| GET | `/registered` | Список зарегистрированных участников |
| POST | `/` | Принятие обновлений от Telegram |

## 📊 Метрики Prometheus

Бот автоматически собирает метрики:
- Количество обработанных обновлений
- Время обработки запросов
- Ошибки в работе

Доступны на эндпоинте `/metrics`

## 🔐 Безопасность

- Не коммитьте реальные значения в `.env` файл
- Используйте `.env.example` как шаблон
- Регулярно обновляйте зависимости
- Подключите двухфакторную аутентификацию Telegram аккаунта

## 🤝 Контрибьютинг

1. Создайте новую ветку: `git checkout -b feature/your-feature`
2. Сделайте изменения и коммитьте: `git commit -am 'Add feature'`
3. Пушьте в ветку: `git push origin feature/your-feature`
4. Откройте Pull Request

## 📝 Лицензия

MIT

## 👤 Автор

[ilyakasharokov](https://github.com/ilyakasharokov)
