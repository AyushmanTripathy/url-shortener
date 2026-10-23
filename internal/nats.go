package internal

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NATSConnection struct {
	nc *nats.Conn
	js jetstream.JetStream

	ctx    context.Context
	cancel context.CancelFunc
	cc     jetstream.ConsumeContext
}

func (conn *NATSConnection) Init() {
	var err error
	connectionUrl, found := os.LookupEnv("NATS_URL")
	if !found {
		log.Fatal("NATS_URL not found")
	}

	conn.nc, err = nats.Connect(connectionUrl)
	if err != nil {
		log.Fatal("Failed to connect to NATS", err)
	}

	conn.js, err = jetstream.New(conn.nc)
	if err != nil {
		log.Fatal("Failed to create Jetstream")
	}

}

func (conn *NATSConnection) EnsureStream(name string, subjects []string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, err := conn.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
	})

	if err != nil {
		log.Fatal("Stream init failed", err)
	}

	log.Println("Stream", name, "init success")
}

func (conn *NATSConnection) EnsureConsumer(name string, stream string, subject string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, err := conn.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:           name,
		FilterSubject:     subject,
		InactiveThreshold: 72 * time.Hour,
		DeliverPolicy:     jetstream.DeliverNewPolicy,
	})

	if err != nil {
		log.Fatal("Consumer init failed", err)
	}

	log.Println("Consumer", name, "init success")
}

func (conn *NATSConnection) Receive(consumerName string, streamName string, handler jetstream.MessageHandler) {
	conn.ctx, conn.cancel = context.WithTimeout(context.Background(), time.Second*30)

	stream, err1 := conn.js.Stream(conn.ctx, streamName)
	if err1 != nil {
		log.Fatal("Failed to retreive stream", err1)
	}

	consumer, err2 := stream.Consumer(conn.ctx, consumerName)
	if err2 != nil {
		log.Fatal("Failed to create stream", err2)
	}

	conn.cc, _ = consumer.Consume(handler)
}

func (conn *NATSConnection) Emit(subjectName string, payload []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, err := conn.js.Publish(ctx, subjectName, payload)
	return err
}

func (conn *NATSConnection) CloseReceiver() {
	conn.cancel()
	conn.cc.Stop()
}
