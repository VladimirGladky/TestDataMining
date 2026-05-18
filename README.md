# TestDataMining — парсер lenta.com

Парсер каталога **lenta.com** на Go. Бьёт по внутреннему JSON API
`POST https://lenta.com/api-gateway/v1/catalog/items/selections` через прокси.

Lenta стоит за Qrator (российский аналог Cloudflare с JS-челленджем), поэтому
сессия инициализируется один раз через headless Chrome (chromedp): он проходит
челлендж, забирает cookies + `sessiontoken` и передаёт обычному `http.Client`,
который дальше собирает данные напрямую. Никаких ручных копирований из DevTools.

## Возможности

- Автоматический обход Qrator headless Chrome'ом через тот же прокси.
- Выбор конкретного магазина по подстроке адреса перед сбором — цены/наличие соответствуют.
- Идёт по выбранным категориям (`selectionId`) с пагинацией до конца.
- Сохраняет: `name`, `price`, `price_regular`, `url`, единицу продажи, рейтинг.
- Retry с экспоненциальным backoff, случайные паузы между страницами.

## Требования

- Go ≥ 1.22
- Google Chrome или Chromium в системе (для headless bootstrap)
- HTTP/HTTPS прокси

## Быстрый старт

```bash
cp .env.example .env
# заполнить PROXY_URL, LENTA_STORE, LENTA_SELECTION_IDS
go run . -out out/products.json
```

## Локальный прокси через tinyproxy

Для тестового запуска подойдёт локальный tinyproxy:

```bash
brew install tinyproxy
tinyproxy -c scripts/tinyproxy.conf            # 127.0.0.1:3128
# в .env: PROXY_URL=http://127.0.0.1:3128
```

## Конфигурация (.env)

| Переменная             | Назначение                                                       |
| ---------------------- | ---------------------------------------------------------------- |
| `PROXY_URL`            | HTTP/HTTPS прокси. Через него идут и chromedp, и API-запросы.    |
| `LENTA_STORE`          | Подстрока в адресе/названии/alias магазина (`Можайское ш., 39`, `ТК639`, `0639`). Должна однозначно опознать один магазин. |
| `LENTA_SELECTION_IDS`  | ID категорий через запятую (из URL: `frukty-310002385` → `310002385`). |
| `LENTA_DOMAIN`         | Город. По умолчанию `moscow`.                                    |
| `LENTA_DELIVERY_MODE`  | `pickup` или `delivery`. По умолчанию `pickup`.                  |
| `LENTA_PAGE_LIMIT`     | Размер страницы. По умолчанию 40.                                |

Опциональные поля видны в `.env.example`.

## Как это работает

1. `Bootstrap`: chromedp запускает headless Chrome с `--proxy-server=$PROXY_URL`,
   открывает `lenta.com/`. Qrator отдаёт JS-челлендж → браузер его проходит →
   `POST __qrator/validate` → 200, после чего сайт начинает работать.
2. Из браузера достаются: cookies (`UserSessionId`, `Utk_DvcGuid`,
   `qrator_jsr`, `qrator_jsid`, …), `SessionToken` из ответа
   `POST /api/rest/sessionGet`. Chrome закрывается.
3. `FindStore` через обычный `http.Client` (тот же прокси) дёргает
   `POST /api-gateway/v1/stores/pickup/search` с пустым телом, ищет магазин с
   совпадением подстроки в адресе/title/alias.
4. `SelectPickupStore` — `PUT /api-gateway/v1/stores/pickup/{storeId}`
   привязывает сессию к выбранной точке.
5. Для каждой `selectionId` пагинируется
   `POST /api-gateway/v1/catalog/items/selections`, маппится в `Product`.
6. `output.WriteJSON` пишет итог в `out/products.json`.

## Структура вывода

`out/products.json`:

```json
{
  "generated_at": "2026-05-18T01:09:25Z",
  "source": "lenta.com",
  "categories": [
    {
      "id": 310002385,
      "name": "Фрукты",
      "products": [
        {
          "id": 94429,
          "name": "Лимоны, весовые",
          "url": "https://lenta.com/product/limony-ves-94429/",
          "price": 53,
          "price_regular": 69.47,
          "has_discount": true,
          "unit": "кг",
          "unit_quantity": 0.2,
          "rating": 4.8,
          "rating_votes": 13036,
          "category_id": 310002385,
          "category_name": "Фрукты"
        }
      ]
    }
  ]
}
```

## Структура проекта

```
.
├── main.go                          # точка входа: bootstrap → store-select → scrape
├── internal/
│   ├── config/config.go             # .env, валидация
│   ├── lenta/
│   │   ├── session.go               # headless-bootstrap через chromedp
│   │   ├── client.go                # HTTP-клиент, заголовки, retry
│   │   ├── stores.go                # поиск и выбор магазина
│   │   ├── types.go                 # типы запроса/ответа Lenta API
│   │   └── scraper.go               # пагинация, маппинг в Product
│   └── output/json.go               # запись JSON
└── scripts/tinyproxy.conf           # минимальный локальный прокси
```
