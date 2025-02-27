# cleanup
Это приложение на Go предназначено для очистки указанных папок от старых файлов. Программа ищет самый свежий файл в каждой папке (сравнивая время создания и время модификации), вычисляет день отсечки, отступая назад на заданное количество дней от этой даты, и удаляет файлы, у которых и время создания, и время модификации старше дня отсечки.

## Функциональность

- **Аргументы командной строки:**
  - Первый аргумент:
    - Если является числом, то интерпретируется как количество дней, на которое нужно отступить от даты самого свежего файла в папке для вычисления дня отсечки.
    - Если не число, то считается путём к YAML файлу конфигурации.
  - Остальные аргументы – список папок для очистки.

- **Чтение параметров из переменных окружения:**
  - `DAYS` — количество дней (целое число).
  - `FOLDERS` — список папок для очистки, разделённых запятой.

- **Логирование:**
  - После выполнения скрипт создаёт (или обновляет) файл `cleanup.log`, в котором записываются:
    - Время запуска.
    - Количество обнаруженных файлов.
    - Количество удалённых файлов.

## Примеры использования

### Запуск с аргументами командной строки

Чтобы удалить файлы в папках \\network\share\folder1 и \\network\share\folder2, где отсечка считается от самого свежего файла минус 10 дней:

```bash
./cleanup 10 \\network\share\folder1 \\network\share\folder2
```

## Запуск с YAML конфигурацией

Создайте YAML файл (например, config.yml):

```yaml
Копировать
days: 10
folders:
  - "\\network\\share\\folder1"
  - "\\network\\share\\folder2"
```

Запустите приложение, передав путь к файлу:

```bash
./cleanup config.yml
```

### Использование переменных окружения

Можно задать параметры через переменные окружения:

```bash
export DAYS=10
export FOLDERS="\\network\\share\\folder1,\\network\\share\\folder2"
./cleanup
```

## Планирование задач

Приложение можно запускать по планировщику задач (cron для Linux или Планировщик задач Windows).

### Пример для cron (Linux)

Добавьте в crontab, например:

```cron
0 2 * * * /path/to/cleanup 10 /mnt/network/folder1 /mnt/network/folder2
```

### Пример для Планировщика задач (Windows)

Создайте задачу, которая будет запускать:

```bat
C:\path\to\cleanup.exe 10 \\network\share\folder1 \\network\share\folder2
```