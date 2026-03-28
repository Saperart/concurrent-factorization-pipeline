# Concurrent Factorization Pipeline

> High-performance concurrent service for prime factorization of integers using a pipeline architecture with separation of computation and I/O stages.

> **Note:** The implementation source code is not publicly available due to course confidentiality and academic integrity requirements.

---

## Overview

Проект представляет собой конкурентный сервис факторизации чисел, реализованный как **pipeline** с разделением этапов обработки.

Основные требования:
- высокая производительность при обработке большого количества чисел
- корректная работа при конкурентной нагрузке
- управляемый параллелизм
- безопасное завершение обработки при ошибках

---

## Key Features

- **Concurrent processing (goroutines + channels)**  
  Параллельная факторизация чисел с использованием каналов

- **Pipeline architecture**  
  Разделение обработки на независимые стадии (compute / write)

- **Independent worker pools**  
  Отдельные пулы воркеров для CPU-bound и I/O-bound этапов

- **Context cancellation**  
  Поддержка отмены обработки через `context.Context`

- **Error propagation**  
  Раннее завершение всех воркеров при ошибках записи

- **Functional Options pattern**  
  Гибкая настройка количества воркеров

---

## Worker model

- входные данные разбиваются между factorization workers
- результаты передаются через каналы
- write workers обрабатывают поток результатов и записывают их в `io.Writer`
- используется конвейерная обработка (pipeline)

---

## Concurrency

Реализовано:

- использование **goroutines** для параллельной обработки
- синхронизация через **channels** без применения mutex
- разделение CPU-bound и I/O-bound стадий
- безопасная остановка через `context.Context`

