# Intent-Agent: Техническое задание и план продвижения

**Продукт:** Open-source агент мониторинга интент-сигналов и лидогенерации
**Стек:** Go (агент/бэкенд) + современный веб-UI
**Цель проекта:** репутация в опенсорсе → аудитория → путь к open-core продукту
**Рабочие названия (выбрать одно):** `signalhound`, `leadpulse`, `intently`, `scout`, `prospekt`

---

## Часть I. Продукт и позиционирование

### 1.1. Одно предложение (elevator pitch)

> Self-hosted AI-агент, который 24/7 мониторит интернет (HN, Reddit, вакансии, RSS, GitHub) на сигналы покупательского интереса по вашему ICP, скорит их через LLM и присылает готовых лидов в Telegram/Slack — open-source альтернатива AI SDR-платформам за $500+/мес.

### 1.2. Целевые пользователи

| Сегмент | Зачем им это |
|---|---|
| Соло-консультанты и фрилансеры (dev, design, marketing) | Не могут платить $500+/мес за AI SDR, хотят входящий поток |
| Инди-хакеры / основатели микро-SaaS | Ищут ранних пользователей по обсуждениям их проблемы |
| Небольшие агентства и sales-команды | Self-hosted = контроль данных, GDPR, нет vendor lock-in |
| DevRel / комьюнити-менеджеры | Мониторинг упоминаний технологии/бренда с анализом тональности |

### 1.3. Отстройка от конкурентов

- **vs AI SDR SaaS (AiSDR, Artisan, 11x):** бесплатно, self-hosted, свои данные, нет спам-машины — только поиск и скоринг, аутрич остаётся человеку (этичная позиция = часть бренда).
- **vs Clay/Apollo:** не база контактов, а *монитор живых сигналов* — ловим момент, когда боль высказана публично.
- **vs самописные n8n-воркфлоу:** один бинарник/докер-контейнер, продуманный скоринг, UI, история, дедупликация — не надо собирать из 30 нод.

### 1.4. Принципиальные решения (product principles)

1. **Single binary.** Всё в одном бинарнике Go: агент, API, встроенный UI (embed). `docker run` или `./agent serve` — работает.
2. **Config as code.** ICP и источники описываются YAML-файлом — гитуется, шарится, версионируется. UI редактирует тот же конфиг.
3. **BYO-LLM.** Пользователь подключает любой LLM: OpenAI, Anthropic, Ollama (локальный) — через единый интерфейс провайдера.
4. **No outreach automation.** Принципиально не делаем автоотправку сообщений. Агент находит и готовит — человек отправляет. (Защита от ToS-банов, спама и репутационных рисков; проговорить в README как философию.)
5. **Легковесность.** SQLite по умолчанию, Postgres опционально. Работает на VPS за $5.

---

## Часть II. Техническое задание — агент (Go)

### 2.1. Архитектура (высокий уровень)

```
                 ┌─────────────────────────────────────────┐
                 │              Single Go Binary            │
                 │                                          │
  sources ──────▶│  Collectors ──▶ Pipeline ──▶ Store       │
  (HN, Reddit,   │  (per-source    (dedup,      (SQLite/    │
   RSS, jobs,    │   pollers)      enrich,       Postgres)  │
   GitHub)       │                 LLM score)               │
                 │                    │                     │
                 │                    ▼                     │
                 │  Notifier (Telegram/Slack/Webhook/Email) │
                 │                    │                     │
                 │  HTTP API (REST) ◀─┴─▶ Embedded Web UI   │
                 └─────────────────────────────────────────┘
```

### 2.2. Модули и требования

#### 2.2.1. Collectors (сборщики сигналов)

Интерфейс:

```go
type Collector interface {
    Name() string
    Collect(ctx context.Context, since time.Time) ([]RawSignal, error)
}
```

MVP-набор (порядок реализации):

| # | Источник | Способ | Примечания |
|---|---|---|---|
| 1 | Hacker News | Algolia HN Search API (бесплатный, без ключа) | Поиск по keywords из конфига; идеален для старта |
| 2 | Reddit | Официальный API (OAuth) | Настраиваемые сабреддиты + keywords |
| 3 | RSS/Atom | Парсер лент | Универсальный: блоги, releases, Google Alerts RSS |
| 4 | GitHub | REST API: issues/discussions search | «pain in public»: ищем issue с болью по теме |
| 5 | Вакансии | HN Who is hiring + RSS джоб-борд | Вакансия «ищем LLM-инженера» = сильнейший интент-сигнал |

v0.2+: Telegram-каналы (MTProto, userbot — отдельный модуль из-за рисков), Mastodon/Bluesky (открытые API), Stack Overflow, Product Hunt.
**Сознательно не делаем:** LinkedIn и X-скрапинг (ToS-риски; максимум — импорт вручную выгруженных данных).

Требования:
- Каждый коллектор — отдельная горутина с индивидуальным расписанием (cron-выражение в конфиге).
- Rate limiting per-source (golang.org/x/time/rate), экспоненциальный backoff, respect robots/API limits.
- Инкрементальность: хранить курсор `last_seen` per source, не перечитывать старое.

#### 2.2.2. Pipeline (обработка)

Этапы (каждый — Stage с интерфейсом, соединяются в конвейер по каналам):

1. **Normalize** — приведение RawSignal к единой модели Signal.
2. **Dedup** — по content-hash + fuzzy (одинаковый пост в 2 источниках). Хранить хэши в store.
3. **Pre-filter** — дешёвый фильтр до LLM: keywords include/exclude, языки, минимальная длина. Цель — резать 80% мусора бесплатно.
4. **LLM Scoring** — главная ценность. Промпт собирается из ICP-конфига пользователя. Выход — строгий JSON (см. 2.4).
5. **Enrich (опционально, v0.2)** — подтянуть данные об авторе/компании (сайт компании через Firecrawl-совместимый API, публичные данные). Плагинный интерфейс Enricher.
6. **Route** — score ≥ threshold → уведомление; иначе — просто в базу (видно в UI как «лента»).

#### 2.2.3. LLM-провайдер

```go
type LLMProvider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
```

- Реализации: OpenAI-совместимый (закрывает OpenAI/OpenRouter/vLLM/Ollama), Anthropic.
- Структурированный вывод: JSON schema + валидация + один авторетрай при невалидном JSON.
- Подсчёт токенов и стоимости per-request; дневной бюджет-лимит в конфиге (`llm.daily_budget_usd`) — при превышении pipeline останавливает скоринг и шлёт алерт.
- Кэш скоринга по content-hash (не скорить одно дважды).

#### 2.2.4. Модель данных (ядро)

```go
type Signal struct {
    ID          string    // ulid
    Source      string    // "hackernews", "reddit:r/golang"
    URL         string
    Author      string
    Title       string
    Content     string
    PublishedAt time.Time
    CollectedAt time.Time
    ContentHash string
}

type Score struct {
    SignalID   string
    Value      int      // 0-100
    IntentType string   // "buying" | "pain" | "hiring" | "research" | "none"
    Reasons    []string // объяснимость — обязательна
    ICPMatch   []string // какие критерии ICP совпали
    DraftReply string   // черновик первого касания (не отправляется!)
    Model      string
    CostUSD    float64
}

type Lead struct { // сгруппированные сигналы одного автора/компании
    ID        string
    Name      string
    Company   string
    Signals   []string // signal IDs
    BestScore int
    Status    string   // "new" | "reviewed" | "contacted" | "ignored" | "customer"
    Notes     string
}
```

#### 2.2.5. Конфигурация (config.yaml — публичный контракт продукта)

```yaml
icp:
  description: >
    Компании 20–500 человек с продуктом, внедряющие LLM (RAG, агенты),
    без сильной внутренней ML-команды. Регионы: EU, US. Языки: en, ru.
  positive_signals:
    - "жалуются на качество/стоимость LLM в проде"
    - "ищут LLM/AI-инженера или консультанта"
  negative_signals:
    - "студенческие вопросы"
    - "крупные корпорации с ML-отделами"

sources:
  hackernews:
    enabled: true
    schedule: "*/30 * * * *"
    keywords: ["RAG production", "LLM cost", "hiring LLM engineer"]
  reddit:
    subreddits: ["LocalLLaMA", "MachineLearning", "golang"]
    schedule: "0 * * * *"
  rss:
    feeds:
      - "https://example.com/jobs.rss"

llm:
  provider: openai-compatible
  base_url: "http://localhost:11434/v1"   # Ollama
  model: "qwen2.5:14b"
  daily_budget_usd: 2.0

scoring:
  notify_threshold: 70

notifiers:
  telegram:
    chat_id: "..."
  webhook:
    url: "https://..."
```

#### 2.2.6. API (REST, chi или stdlib mux)

- `GET /api/signals?score_gte=&source=&period=` — лента с фильтрами, пагинация
- `GET /api/leads`, `PATCH /api/leads/{id}` — статусы, заметки
- `GET /api/stats` — счётчики: сигналов/день, стоимость LLM, распределение скоров, конверсия по источникам
- `GET/PUT /api/config` — чтение/запись конфига (валидация + hot reload)
- `POST /api/score/test` — «прогнать текст через скоринг» (для отладки ICP-промпта)
- SSE `GET /api/events` — живой стрим новых сигналов для UI
- Auth: один API-token (env), сессия для UI. Multi-user — вне скоупа MVP.

#### 2.2.7. Нефункциональные требования

- Graceful shutdown, structured logging (slog), Prometheus `/metrics`.
- Миграции — golang-migrate или embed goose; SQLite default.
- Тесты: пайплайн и скоринг — с fake LLM; коллекторы — на записанных фикстурах (golden files).
- CI: GitHub Actions — lint (golangci-lint), test, goreleaser (бинарники под linux/mac/win + докер-образ multi-arch).
- Лицензия: **AGPL-3.0** (защита от «SaaS-паразитов» при open-core пути) либо Apache-2.0 (максимум adoption). Рекомендация: AGPL, как у n8n-подобных.

---

## Часть III. Техническое задание — UI

### 3.1. Технологический выбор

**Рекомендация: React (Vite) + TypeScript + Tailwind, собирается в статику и embed-ится в Go-бинарник (`embed.FS`).**
Альтернатива для минимализма — templ + htmx (чисто Go-стек), но: (а) красивый UI — заявленное требование и половина wow-эффекта на скриншотах, (б) React-скиллы полезны вам самому. Дизайн — тёмная тема по умолчанию (дев-аудитория, скриншоты в соцсетях выглядят дорого), аккуратная типографика, один акцентный цвет.

### 3.2. Экраны

1. **Dashboard** — карточки: сигналов сегодня / хотлидов / расход LLM $ / топ-источник; график сигналов по дням; live-лента (SSE) последних скоров.
2. **Feed (главный экран)** — список сигналов: score-бейдж (цвет по величине), источник-иконка, заголовок, сниппет, время; фильтры по скору/источнику/типу интента; клик → карточка сигнала: полный текст, **объяснение скора (reasons, ICP-match)**, черновик ответа с кнопкой Copy, ссылка на оригинал, действия (в лиды / игнор / автора в игнор).
3. **Leads (мини-CRM)** — kanban или таблица: new → reviewed → contacted → customer; заметки; связанные сигналы.
4. **ICP & Sources (настройки)** — форма-редактор конфига с валидацией + «Test scoring»: вставил текст → увидел скор и объяснение (киллер-фича для подбора ICP-промпта).
5. **Stats** — конверсия источников (какой источник даёт хотлиды), стоимость на лид, история бюджета LLM.

### 3.3. UX-требования

- Пустые состояния с подсказками («добавьте первый источник») — важно для первого впечатления в демо.
- Всё интерактивное ≤ 200 мс; тяжёлое — оптимистичные апдейты.
- Онбординг-визард при первом запуске: ICP → источники → LLM-ключ → первый прогон. Цель: «от docker run до первого лида ≤ 10 минут».
- Скриншотогеничность как явное требование: главный экран должен «продавать» проект в README и постах.

### 3.4. Definition of Done для MVP (v0.1)

- [ ] 3 коллектора: HN, Reddit, RSS
- [ ] Pipeline: dedup + pre-filter + LLM-скоринг с объяснением + бюджет-лимит
- [ ] Telegram-уведомления с карточкой лида и черновиком
- [ ] UI: Dashboard, Feed, Settings c test-scoring
- [ ] Single binary + docker-образ, goreleaser, README с гифкой
- [ ] Демо-режим с сид-данными (`--demo`) — чтобы люди смотрели UI без настройки

### 3.5. Роадмап после MVP

| Версия | Фокус |
|---|---|
| v0.2 | GitHub + вакансии коллекторы; enrichment-плагины; Slack/Discord notifier; Postgres |
| v0.3 | Мульти-ICP (несколько профилей поиска); экспорт CSV; webhook в CRM (HubSpot/Pipedrive) |
| v0.4 | Правила/алерты (например «упомянули конкурента»); дайджест-рассылка вместо потока |
| v1.0 | Стабильный API, плагинная система коллекторов, docs-сайт |
| Cloud (опционально, при traction) | Hosted-версия: open-core монетизация |

### 3.6. Оценка трудозатрат (вечера/выходные, реалистично)

- Недели 1–2: каркас, конфиг, HN-коллектор, pipeline без LLM, SQLite
- Недели 3–4: LLM-скоринг + Telegram + Reddit/RSS
- Недели 5–6: API + UI (Dashboard, Feed)
- Недели 7–8: Settings + test-scoring, полировка, README, демо-гифка, goreleaser
- **Итого: ~2 месяца до публичного v0.1.** Резать скоуп, но не качество README и скриншотов.

---

---

*План вывода на рынок вынесен в `docs/private/GO-TO-MARKET.md` — он не публикуется
и намеренно не входит в спеку: `/plan` планирует только продукт.*
