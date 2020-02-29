package pubsub

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/openfaas-incubator/connector-sdk/types"
)

const subID = "openfaas_connector_"

// Broker used to subscribe to NATS subjects
type Broker interface {
	Subscribe(types.Controller, []string) error
}

// NewBroker loops until we are able to connect to the NATS server
func NewBroker(config BrokerConfig) (Broker, error) {
	broker := &broker{}

	for {
		client, err := pubsub.NewClient(context.Background(), config.ProjectID)
		if client != nil && err == nil {
			broker.client = client
			break
		}

		if client != nil {
			client.Close()
		}

		log.Println("Wait for brokers to come up.. ", config.ProjectID)
		time.Sleep(1 * time.Second)
		// TODO Add healthcheck
	}

	return broker, nil
}

type broker struct {
	client *pubsub.Client
}

func (b *broker) topic(ctx context.Context, topic string) (*pubsub.Topic, error) {
	t := b.client.Topic(topic)
	exists, err := t.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("Get Topic: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("Topic doesn't exist")
	}
	return t, nil
}

func (b *broker) subscription(ctx context.Context, id string, topic *pubsub.Topic) (*pubsub.Subscription, error) {
	sub := b.client.Subscription(id)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("Get Subscription: %w", err)
	}
	if exists {
		return sub, nil
	}

	return b.client.CreateSubscription(ctx, id, pubsub.SubscriptionConfig{
		Topic:       topic,
		AckDeadline: 20 * time.Second,
	})
}

func (b *broker) receive(topic string, controller types.Controller) func(ctx context.Context, m *pubsub.Message) {
	return func(ctx context.Context, m *pubsub.Message) {
		controller.InvokeWithContext(ctx, topic, &m.Data)
		m.Ack()
	}
}

func (b *broker) handle(ctx context.Context, sub *pubsub.Subscription, topic string, controller types.Controller) {
	for {
		ctx := context.Background()
		if err := sub.Receive(ctx, b.receive(topic, controller)); err != context.Canceled {
			log.Printf("Received error for topic %s: %v\n", topic, err)
		}
		time.Sleep(time.Second) // don't know if we want this here
	}
}

// Subscribe to a list of NATS subjects and block until interrupted
func (b *broker) Subscribe(controller types.Controller, topics []string) error {
	log.Printf("Configured topics: %v", topics)

	if b.client == nil {
		return fmt.Errorf("client was nil, try to reconnect")
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	ctx := context.Background()
	cctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		b.client.Close()
	}()

	for _, topic := range topics {
		log.Printf("Binding to topic: %q", topic)

		t, err := b.topic(ctx, topic)
		if err != nil {
			return fmt.Errorf("Get Topic: %w", err)
		}

		sub, err := b.subscription(ctx, subID+topic, t)
		if err != nil {
			return fmt.Errorf("Get Subscription: %w", err)
		}

		go b.handle(cctx, sub, topic, controller)
		log.Printf("Subscription: %s ready", sub.String())
	}

	wg.Wait()
	return nil
}
