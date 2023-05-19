package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqptransport "github.com/go-kit/kit/transport/amqp"
	"github.com/kokizzu/gotro/L"
	"github.com/kokizzu/gotro/S"
	"github.com/streadway/amqp"
)

const (
	reconnectDelaySec              = 3
	messageExpirationSeconds int64 = 604800000
	LocalRabbitMqDSN               = "amqp://guest:guest@127.0.0.1:5672"
)

type rmqConfig struct {
	dsn        string
	exchange   string
	queue      string
	routingKey string
	mtls       *tls.Config
}

func errLabeller(label string) func(error) error {
	return func(err error) error {
		return fmt.Errorf("%s: %w", label, err)
	}
}

func SubscribeRetrier(ctx context.Context, cfg rmqConfig, handler *amqptransport.Subscriber) error {
	L.Print("SubscribeRetrier: dial amqp...")
	wrapErr := errLabeller(`main.SubscribeRetrier`)
	conn, err := dialer(cfg)
	if L.IsError(err, `dialer`) {
		return wrapErr(err)
	}
	defer func() {
		errConnClose := conn.Close()
		L.IsError(errConnClose, `conn.Close`)
	}()
	ch, err := conn.Channel()
	if L.IsError(err, `conn.Channel`) {
		return wrapErr(err)
	}
	if cfg.dsn == LocalRabbitMqDSN {
		err = ch.ExchangeDeclare(
			cfg.exchange,
			amqp.ExchangeTopic,
			true,
			false,
			false,
			false,
			nil,
		)
		if L.IsError(err, `ch.ExchangeDeclare`) {
			return wrapErr(err)
		}
	}
	q, err := ch.QueueDeclare(
		cfg.queue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-expires": messageExpirationSeconds,
		},
	)
	if L.IsError(err, `ch.QueueDeclare`) {
		return wrapErr(err)
	}
	err = ch.QueueBind(
		q.Name,
		cfg.routingKey,
		cfg.exchange,
		false,
		nil,
	)
	if L.IsError(err, `ch.QueueBind`) {
		return wrapErr(err)
	}
	L.Print("start consume", q.Name)
	messages, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if L.IsError(err, `ch.Consume`) {
		return wrapErr(err)
	}
	L.Print("SubscribeRetrier: start processing amqp status messages")
	processDeliveryFn := handler.ServeDelivery(ch)
	channelNotify := ch.NotifyClose(make(chan *amqp.Error))
	for {
		select {
		case msg := <-messages:
			processDeliveryFn(&msg)

			if err := msg.Ack(false); err != nil {
				L.IsError(err, "msg.Ack: acknowledge could not be delivered to the channel")
			} else {
				L.Print("message was acknowledged:", string(msg.Body))
			}
		case err, ok := <-channelNotify:
			L.Print("SubscribeRetrier: channelNotify closed:", err, ok)
			return wrapErr(err)
		case <-ctx.Done():
			L.Print("SubscribeRetrier: stop processing amqp status messages")
			return nil
		}
	}
}

func dialer(cfg rmqConfig) (conn *amqp.Connection, err error) {
	wrapErr := errLabeller(`main.dialer`)
	if cfg.mtls != nil {
		conn, err = amqp.DialTLS(cfg.dsn, cfg.mtls)
		if L.IsError(err, `amqp.DialTLS`) {
			return nil, wrapErr(err)
		}
	} else {
		conn, err = amqp.Dial(cfg.dsn)
		if L.IsError(err, `amqp.Dial`) {
			return nil, wrapErr(err)
		}
	}
	return conn, nil
}

func PublishRetrier(ctx context.Context, cfg rmqConfig) error {
	L.Print("PublishRetrier: dial amqp...")
	wrapErr := errLabeller(`main.PublishRetrier`)
	conn, err := dialer(cfg)
	if L.IsError(err, `dialer`) {
		return wrapErr(err)
	}
	defer func() {
		err := conn.Close()
		L.IsError(err, `conn.Close`)
	}()

	ch, err := conn.Channel()
	if L.IsError(err, `conn.Channel`) {
		return wrapErr(err)
	}
	defer func() {
		err := ch.Close()
		L.IsError(err, `ch.Close`)
	}()

	channelNotify := ch.NotifyClose(make(chan *amqp.Error))
	for {
		ticker := time.NewTicker(time.Second * reconnectDelaySec)
		select {
		case <-ctx.Done():
			L.Print("PublishRetrier: stop publishing amqp status messages")
			return nil
		case err, ok := <-channelNotify:
			L.Print("PublishRetrier: channelNotify closed:", err, ok)
			return wrapErr(err)
		case <-ticker.C:
			body := S.RandomCB63(1)
			err := ch.Publish(cfg.exchange, cfg.routingKey, false, false, amqp.Publishing{
				Body: []byte(body),
			})
			if L.IsError(err, `ch.Publish`) {
				L.Print("PublishRetrier: failed to publish amqp status messages")
			} else {
				L.Print("PublishRetrier: publish amqp status message body:", body)
			}
		}
	}

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	dsn := os.Getenv(`RMQ_DSN`) // RMQ_DSN=amqp://guest:guest@127.0.0.1:5672 go run main.go
	if dsn == `` {
		panic(`missing RQM_DSN env var`)
	}
	var mtlsConfig *tls.Config
	cert := os.Getenv(`CERT`)
	key := os.Getenv(`KEY`)
	caCert := os.Getenv(`CA_CERT`)
	if cert != `` && key != `` && caCert != `` {
		var err error
		mtlsConfig, err = MtlsConfig(cert, key, caCert)
		L.PanicIf(err, `MtlsConfig`)
	}

	target := rmqConfig{
		dsn:        dsn,
		exchange:   "exchange1",
		queue:      "queue1",
		routingKey: "routingKey1",
		mtls:       mtlsConfig,
	}

	handler := amqptransport.NewSubscriber(
		func(_ context.Context, req any) (resp any, err error) {
			// handler, do echo for now
			return req, nil
		},
		func(_ context.Context, delivery *amqp.Delivery) (any, error) {
			// decoder
			return delivery.Body, nil
		},
		func(_ context.Context, publisher *amqp.Publishing, resp any) error {
			// encoder
			publisher.Body = []byte(fmt.Sprint(resp))
			return nil
		},
	)

	// capture exit signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	go func() {
		ticker := time.NewTicker(time.Second * reconnectDelaySec)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := PublishRetrier(ctx, target)
				L.IsError(err, `PublishRetrier`)
			}
		}
	}()

	ticker := time.NewTicker(time.Second * reconnectDelaySec)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := SubscribeRetrier(ctx, target, handler)
			L.IsError(err, `SubscribeRetrier`)
		}
	}
}
