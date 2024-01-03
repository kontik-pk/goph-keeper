# goph-keeper

GophKeeper представляет собой клиент-серверную систему, позволяющую пользователю надёжно и безопасно хранить логины, 
пароли, бинарные данные и прочую приватную информацию.

## Общее устройство механизма

- Клиент распространяется в виде CLI-приложения;
- В качестве хранилища данных используется PostgreSQL;
- Клиент и сервер обмениваются данными по HTTP-протоколу;
- Чувствительные данные хранятся в зашифрованном виде;
- Механизм конфигурируется через следующие переменные окружения:
  - `POSTGRES_HOST` - хост хранилища
  - `POSTGRES_PORT` - порт хранилища
  - `POSTGRES_USER` - пользователь `goph-keeper`
  - `POSTGRES_PASSWORD` - пароль пользователя `goph-keeper`
  - `POSTGRES_DB` - имя базы данных, в которой хранится вся пользовательская информация;
  - `APPLICATION_PORT` - порт приложения `goph-keeper`
  - `APPLICATION_HOST` - хост приложения `goph-keeper`
  - `KEEPER_ENCRYPTION_KEY` - ключ для шифрования чувствительной информации
- В хранилище `goph-keeper` существуют следующие системные таблицы:
  - `registered_users` - таблица пользователей, зарегистрированных в `goph-keeper`
  - `credentials` - таблица с сохраненными логинами/паролями пользователей. Каждый пользователь
    через приложение может получить только свои логины/пароли. Пароли хранятся в зашифрованном виде
  - `notes` - таблица, в которой хранится произвольная пользовательская информация - различные
заметки, бинарные данные etc. Все содержимое хранится в зашифрованном виде. Каждый пользователь
через приложение может получить только свои данные
  - `cards` - данные банковских карт: имя банка, номер карты, cv-код, пароль от банковского приложения.
CV и пароли хранятся в зашифрованном виде. Каждый пользователь через приложение может получить данные
только своих карт

## Cхема взаимодействия с системой

**Для нового пользователя**:

- Пользователь получает клиент под необходимую ему платформу
- Пользователь проходит процедуру первичной регистрации
- Пользователь добавляет в клиент новые данные
- Клиент синхронизирует данные с сервером

**Для существующего пользователя**:

- Пользователь получает клиент под необходимую ему платформу
- Пользователь проходит процедуру аутентификации
- Клиент синхронизирует данные с сервером
- Пользователь запрашивает данные
- Клиент отображает данные для пользователя

## Установка приложения для своей платформы

- Склонировать репозиторий
- Собрать приложение

    ```shell
    make install
    ```

- Остановить приложение

    ```shell
    make stop
    ```
  
Команда `make install` поднимает PostgreSQL в докере, применяет необходимые для работы сервиса миграции,
а затем запускает HTTP сервер приложения, который начинает принимать запросы.

## Возможности приложения

TLDR: для каждой команды доступна справка с примерами использования.

**Посмотреть дату сборки приложения**

```shell
goph-keeper build-date
```

**Посмотреть версию приложения**

```shell
goph-keeper --version
```

**Регистрация в приложении**

```shell
goph-keeper register --login <user-system-login> --password <user-system-password>
```

**Вход в приложение**

```shell
goph-keeper login --login <user-system-login> --password <user-system-password>
```

**Добавить данные о банковской карте**

```shell
goph-keeper  add-card --user <user-system-login> --bank <bank-name> --number <card-number> --cv <card-cv> --password <password>
```

Номер карты должен содержать 16 знаков, cv - 3 знака. Можно добавить метаинформацию о карте:

```shell
goph-keeper  add-card --user <user-system-login> --bank <bank-name> --number <card-number> --cv <card-cv> --password <password> --metadata <some metadata>
```

**Добавить логин/пароль**

```shell
goph-keeper add-credentials --user <user-name> --login <user-login> --password <password to store> --metadata <some description>
```

**Добавить произвольную текстовую информацию**

```shell
goph-keeper add-note --user <user-name> --title <note title> --content <note content> --metadata <note metadata>
```

**Удалить логин/пароль**

```shell
goph-keeper delete-credentials --user <user-name> --login <user-login>
```

Можно удалить все сохраненные пары логин/пароль для пользователя, не указывая конкретный логин:

```shell
goph-keeper delete-credentials --user <user-name>
```

**Удалить произвольную информацию**

```shell
goph-keeper delete-note --user <user-name> --title <note title>
```

Можно удалить все данные для пользователя, если не указывать идентификатор данных:

```text
goph-keeper delete-note --user <user-name>
```

**Удалить данные банковских карт**

Эта команда удалит все данные карт банка `<bank-name>` пользователя `<user-name>`

```shell
goph-keeper  delete-card --user <user-name> --bank <bank-name>
```

Эта команда удалит данные карты с номером `<card-number>` пользователя `<user-name>`

```shell
goph-keeper  delete-card --user <user-name> --number `<card-number>`
```

**Получить сохраненные пары логин/пароль**

```shell
goph-keeper get-credentials --user <user-name>
```

Можно получить информацию для конкретного логина:

```text
goph-keeper get-credentials --user <user-name> --login <login>
```

**Получить сохраненные произвольные данные**

```shell
goph-keeper get-note --user <user-name>
```

Можно получить информацию для конкретного идентификатора:

```text
goph-keeper get-credentials --user <user-name> --title <note title>
```

**Получить данные сохраненных банковских карт**

```shell
goph-keeper get-card --user <user-name>
```

Можно получить информацию по картам конкретного банка:

```text
goph-keeper get-credentials --user <user-name> --bank <bank-name>
```

Можно получить информацию по карте с конкретным номером:

```text
goph-keeper get-credentials --user <user-name> --number <card-number>
```

**Изменить пароль для сохраненного логина**

```text
goph-keeper update-credentials --user <user-name> --login <saved-login> --password <new-password>
```

**Отредактировать сохраненные произвольные данные**

```text
goph-keeper update-notes --user <user-name> --title <note-title> --content <new-content>
```
