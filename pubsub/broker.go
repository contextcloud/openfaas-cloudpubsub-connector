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

const subID = "openfaas_pubsub_worker_group"

// Broker used to subscribe to NATS subjects
type Broker interface {
	Subscribe(types.Controller, []string) error
}

type broker struct {
	client *pubsub.Client
}

// NewBroker loops until we are able to connect to the NATS server
func NewBroker(config BrokerConfig) (Broker, error) {
	broker := &broker{}

	for {
		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, config.ConnTimeout)

		client, err := pubsub.NewClient(ctx, config.ProjectID)
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

// Subscribe to a list of NATS subjects and block until interrupted
func (b *broker) Subscribe(controller types.Controller, topics []string) error {
	log.Printf("Configured topics: %v", topics)

	if b.client == nil {
		return fmt.Errorf("client was nil, try to reconnect")
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	ctx := context.Background()

	subs := []*pubsub.Subscription{}
	for _, topic := range topics {
		log.Printf("Binding to topic: %q", topic)

		t := b.client.Topic(topic)
		ok, err := t.Exists(ctx)
		if err != nil {
			return fmt.Errorf("Get Topic: %w", err)
		}
		if !ok {
			return fmt.Errorf("Topic doesn't exist")
		}

		sub, err := b.client.CreateSubscription(ctx, subID, pubsub.SubscriptionConfig{
			Topic:       t,
			AckDeadline: 20 * time.Second,
		})
		if err != nil {
			return fmt.Errorf("CreateSubscription: %w", err)
		}

		if err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
			controller.InvokeWithContext(ctx, topic, &m.Data)
			m.Ack()
		}); err != nil {
			return fmt.Errorf("Receive: %w", err)
		}

		subs = append(subs, sub)
	}

	for _, sub := range subs {
		log.Printf("Subscription: %s ready", sub.String())
	}

	wg.Wait()
	b.client.Close()
	return nil
}
