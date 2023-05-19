
# rabbit1

testing rabbitmq publish, subscribe, reconnect

```
RMQ_DSN=amqp://guest:guest@127.0.0.1:5672 go run .
```

other env options:
- `CERT` (string)
- `KEY` (string)
- `CA_CERT` (string)
- `PUBLISH_INTERVAL_US` (int, microsecond delay between publish)
- `DEBUG` (bool, print ok/failed publish/subscribe)

note:
- est rps - estimated rps, excluding delay between publish or subscribe
- real rps - real rps, including delay between publish or subscribe