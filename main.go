package main

import (
	"log"

	"github.com/openfaas-incubator/connector-sdk/types"

	"github.com/contextcloud/openfaas-cloudpubsub-connector/config"
	"github.com/contextcloud/openfaas-cloudpubsub-connector/pubsub"
)

func main() {
	creds := types.GetCredentials()
	config := config.Get()

	controllerConfig := &types.ControllerConfig{
		UpstreamTimeout:          config.UpstreamTimeout,
		GatewayURL:               config.GatewayURL,
		RebuildInterval:          config.RebuildInterval,
		PrintResponse:            config.PrintResponse,
		PrintResponseBody:        config.PrintResponseBody,
		TopicAnnotationDelimiter: config.TopicAnnotationDelimiter,
		AsyncFunctionInvocation:  config.AsyncFunctionInvocation,
		PrintSync:                config.PrintSync,
	}
	controller := types.NewController(creds, controllerConfig)
	controller.BeginMapBuilder()

	brokerConfig := pubsub.BrokerConfig{
		ProjectID:   config.BrokerProjectID,
		ConnTimeout: config.UpstreamTimeout,
	}

	broker, err := pubsub.NewBroker(brokerConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = broker.Subscribe(controller, config.Topics)
	if err != nil {
		log.Fatal(err)
	}
}
