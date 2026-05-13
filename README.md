# TestDataMining — парсер lenta.com

Парсер каталога **lenta.com** на Go. Бьёт по внутреннему JSON API
`POST https://lenta.com/api-gateway/v1/catalog/items/selections`,
работает в чистом серверном режиме (без headless-браузера), через прокси.

## Возможности

- Идёт по выбранным категориям (`selectionId`) с пагинацией до конца.
- Сохраняет: `name`, `price`, `price_regular`, `url`, единицу продажи, рейтинг,
  привязку к категории.
- Автоматический retry с экспоненциальным backoff и джиттером.
- Между страницами случайная пауза 0.8–1.7 с — чтобы не словить rate-limit.
- Прокси задаётся через ENV `PROXY_URL`.

## Быстрый старт

```bash
cp .env.example .env
# заполнить .env (см. ниже)
go run . -out out/products.json
```

## Локальный прокси через tinyproxy

Для тестового запуска / демонстрации требование «обязательно через прокси»
можно закрыть локальным tinyproxy. Для продакшн-парсинга нужен внешний RU IPv4
(см. секцию «Замечания по антибот-защите» ниже).

```bash
brew install tinyproxy
tinyproxy -c scripts/tinyproxy.conf            # стартует в фоне, порт 127.0.0.1:3128
curl -x http://127.0.0.1:3128 -I https://lenta.com/   # должен вернуть 200
# в .env:
# PROXY_URL=http://127.0.0.1:3128
```

Остановить:
```bash
pkill tinyproxy
```

## Конфигурация (.env)

| Переменная             | Назначение                                                       |
| ---------------------- | ---------------------------------------------------------------- |
| `PROXY_URL`            | HTTP/HTTPS прокси, через который идут все запросы. Обязательно.  |
| `LENTA_COOKIE`         | Полная строка cookie из браузерной сессии lenta.com.             |
| `LENTA_DEVICE_ID`      | Заголовок `deviceid` / `x-device-id` (UUID).                     |
| `LENTA_SESSION_TOKEN`  | Заголовок `sessiontoken` (32 hex-символа).                       |
| `LENTA_USER_SESSION_ID`| Заголовок `x-user-session-id` (UUID).                            |
| `LENTA_SELECTION_IDS`  | ID категорий через запятую. См. ниже.                            |
| `LENTA_DOMAIN`         | Город. По умолчанию `moscow`.                                    |
| `LENTA_DELIVERY_MODE`  | `pickup` или `delivery`. По умолчанию `pickup`.                  |
| `LENTA_PAGE_LIMIT`     | Размер страницы. По умолчанию 40 (как у сайта).                  |

## Откуда взять значения для .env

Все идентификаторы Lenta привязывает к выбранному адресу доставки/магазину.
Чтобы их получить:

1. В Chrome открыть **инкогнито** → `https://lenta.com/`.
2. Выбрать город и **конкретный магазин/адрес доставки**. Это критично —
   именно от него зависят цены и наличие.
3. Открыть DevTools (`Cmd+Opt+I`) → вкладка **Network** → фильтр **Fetch/XHR**.
4. Перейти в любую категорию каталога. В списке появится запрос
   `POST .../api-gateway/v1/catalog/items/selections`.
5. Кликнуть по нему → вкладка **Headers**:
   - `cookie:` целиком → в `LENTA_COOKIE`
   - `deviceid:` → в `LENTA_DEVICE_ID`
   - `sessiontoken:` → в `LENTA_SESSION_TOKEN`
   - `x-user-session-id:` → в `LENTA_USER_SESSION_ID`
6. Вкладка **Payload** → поле `selectionId` → в `LENTA_SELECTION_IDS`.
   Чтобы парсить несколько категорий — добавьте через запятую,
   обходя нужные категории на сайте и собирая их `selectionId`.

Сессия живёт несколько часов; если запросы вдруг начнут возвращать 401/403 —
перезайти на сайт, обновить значения в `.env`.

## Структура вывода

`out/products.json`:

```json
{
  "generated_at": "2026-05-10T15:00:00Z",
  "source": "lenta.com",
  "total_count": 48,
  "categories": [
    {
      "id": 310002385,
      "name": "Фрукты",
      "products": [
        {
          "id": 94429,
          "name": "Лимоны, весовые",
          "url": "https://lenta.com/product/limony-ves-94429/",
          "price": 52.0,
          "price_regular": 52.0,
          "has_discount": false,
          "unit": "кг",
          "unit_quantity": 0.2,
          "rating": 4.8,
          "rating_votes": 12877,
          "category_id": 310002385,
          "category_name": "Фрукты"
        }
      ]
    }
  ]
}
```

Цены — в рублях (API отдаёт в копейках; делим на 100).
Для весовых товаров `price` — это цена за `unit_quantity` `unit` (например,
0.2 кг). Цена за килограмм / штуку находится в `prices.cost` исходного API
и при необходимости легко добавляется в выходную структуру.

## Структура проекта

```
.
├── main.go                          # точка входа
├── internal/
│   ├── config/config.go             # парсинг .env, валидация
│   ├── lenta/
│   │   ├── client.go                # HTTP-клиент, заголовки, retry
│   │   ├── types.go                 # типы запроса/ответа Lenta API
│   │   └── scraper.go               # пагинация, маппинг в Product
│   └── output/json.go               # запись JSON
└── out/                             # игнорируется git
```

## Замечания по антибот-защите

Перед `lenta.com` стоит **Qrator Labs** (видно по `Server: QRATOR`).
Простые curl-запросы со всеми скопированными заголовками проходят 200 OK,
но если сессионные cookies протухли — Qrator может отдать challenge-страницу.
В таком случае обновите `.env` свежими значениями из браузера.

Если запросы начнут массово 403/429:
- увеличьте паузы (`politeSleep` в `scraper.go`)
- смените прокси
- обновите cookies
