# Homework 3

Два сервиса из прошлого задания: `product-service` (gRPC, `:50051`) и `cart` (HTTP, `:8080`). Корзины и каталог живут в памяти.

При `GET /user/<user_id>/cart` описание каждого товара запрашивается у `product-service` отдельной горутиной, на всю пачку один таймаут. Пустая корзина в каталог не ходит. Если товара уже нет, ответ `412`.

## Запуск и тесты

```
make run-all
make test
make test-race
make test-api
make bench
```

`make test` не собирает интеграционные тесты (`//go:build api`) и race-тест (`//go:build race`). `make test-api` ходит в уже запущенные сервисы.

## Цена синхронизации

Один прогон `go test -bench=. -benchmem -count=1` на AMD Ryzen 7 PRO 3700, Windows, один поток. До — `sync.Mutex` на обе операции. После — `sync.RWMutex` (`RLock` на чтение, `Lock` на вставку) и `atomic.Uint32` на количество.

| Бенчмарк | До, ns/op | До, B/op | До, allocs/op | После, ns/op | После, B/op | После, allocs/op |
|---|---:|---:|---:|---:|---:|---:|
| BenchmarkAddToCart | 20.47 | 0 | 0 | 21.07 | 0 | 0 |
| BenchmarkGetCart | 1580 | 808 | 4 | 1541 | 808 | 4 |

На одном потоке разница в пределах шума прогона: добавление чуть дороже из-за `RLock` и атомика, чтение чуть дешевле. Аллокации не изменились.

`BenchmarkGetProduct`: 40.93 ns/op, 80 B/op, 1 allocs/op. Каталог и раньше был под `RWMutex`, этот замер только после.
