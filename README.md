# SALOMAT Courier Server 0.3 — Koyeb

Это интернет-сервер для первого теста GPS курьеров.

## Развёртывание на Koyeb через GitHub

1. Создайте новый GitHub repository, например:
   salomat-courier-server
2. Загрузите в корень репозитория:
   - main.go
   - go.mod
   - Dockerfile
3. В Koyeb нажмите Create Web Service.
4. Выберите GitHub и подключите созданный репозиторий.
5. Builder: Dockerfile.
6. Instance: Free.
7. Region: Frankfurt (FRA) — рекомендуется для Таджикистана из доступных free-регионов.
8. Exposed port: 8080 / HTTP.
9. Route: /.
10. Environment variable:
    API_KEY = придуманный длинный секретный ключ
11. Deploy.

После успешного Deploy Koyeb даст HTTPS адрес вида:
https://<имя>-<организация>.koyeb.app

Проверка:
https://ВАШ-АДРЕС.koyeb.app/health

Должен вернуться JSON:
{"ok":true,"service":"SALOMAT Courier Server 0.3"}

## API

GET /api/couriers
GET /api/couriers/1/track

POST /api/couriers/1/status
Header:
X-API-Key: <API_KEY>
JSON:
{"status":"busy"}

POST /api/couriers/1/location
Header:
X-API-Key: <API_KEY>
JSON:
{"latitude":38.5598,"longitude":68.7870}

## Важно

Версия 0.3 хранит GPS в памяти сервера.
При перезапуске/усыплении бесплатного Koyeb история маршрутов очищается.
Для теста живого отслеживания это нормально.
После проверки связи подключим постоянную базу данных.
