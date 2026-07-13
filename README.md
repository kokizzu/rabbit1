
# rabbit1

testing rabbitmq publish, subscribe, reconnect

```
make run CMD='env RMQ_DSN=amqp://guest:guest@127.0.0.1:5672 go run .'
```

other env options:
- `CERT` (string)
- `KEY` (string)
- `CA_CERT` (string)
- `PUB_INTERVAL_US` (int, microsecond (µs) delay between publish, default: 100000 = 100ms = real 10 rps)
- `DEBUG` (bool, print ok/failed publish/subscribe, default: false)
- `SUB_COUNT` (int, subscriber count, default: 1)

note:
- est rps - estimated rps, excluding delay between publish or subscribe
- real rps - real rps, including delay between publish or subscribe

## Maintenance checklist

- [x] Go runtime updated to 1.26.5.
- [x] Go dependencies refreshed and module files tidied.
- [x] AMQP client moved to the maintained `amqp091-go` API used by current go-kit.
- [x] `make test` compiles the example without requiring RabbitMQ.
- [x] `make verify-dependency-security` and `make vulncheck` check dependency security.
