package pubsub

import "time"

// BrokerConfig high level config for the broker
type BrokerConfig struct {
	// ProjectID for Cloud Pub/Sub
	ProjectID string

	// ConnTimeout is the timeout for Dial on a connection.
	ConnTimeout time.Duration
}
