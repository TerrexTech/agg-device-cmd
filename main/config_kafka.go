package main

import (
	"fmt"
	"os"

	"github.com/TerrexTech/agg-device-cmd/device"

	"github.com/TerrexTech/go-commonutils/commonutil"
	"github.com/TerrexTech/go-eventspoll/poll"
	"github.com/TerrexTech/go-kafkautils/kafka"
)

func loadKafkaConfig() (*poll.KafkaConfig, error) {
	kafkaBrokers := *commonutil.ParseHosts(
		os.Getenv("KAFKA_BROKERS"),
	)

	cEventGroup := os.Getenv("KAFKA_CONSUMER_EVENT_GROUP")
	cEventQueryGroup := os.Getenv("KAFKA_CONSUMER_EVENT_QUERY_GROUP")
	cEventTopic := os.Getenv("KAFKA_CONSUMER_EVENT_TOPIC")
	cEventQueryTopic := os.Getenv("KAFKA_CONSUMER_EVENT_QUERY_TOPIC")
	pEventQueryTopic := os.Getenv("KAFKA_PRODUCER_EVENT_QUERY_TOPIC")
	pResponseTopic := os.Getenv("KAFKA_PRODUCER_RESPONSE_TOPIC")

	cEventTopic = fmt.Sprintf("%s.%d", cEventTopic, device.AggregateID)
	cEventQueryTopic = fmt.Sprintf("%s.%d", cEventQueryTopic, device.AggregateID)

	kc := &poll.KafkaConfig{
		EventCons: &kafka.ConsumerConfig{
			KafkaBrokers: kafkaBrokers,
			GroupName:    cEventGroup,
			Topics:       []string{cEventTopic},
		},
		ESQueryResCons: &kafka.ConsumerConfig{
			KafkaBrokers: kafkaBrokers,
			GroupName:    cEventQueryGroup,
			Topics:       []string{cEventQueryTopic},
		},

		ESQueryReqProd: &kafka.ProducerConfig{
			KafkaBrokers: kafkaBrokers,
		},
		SvcResponseProd: &kafka.ProducerConfig{
			KafkaBrokers: kafkaBrokers,
		},
		ESQueryReqTopic:  pEventQueryTopic,
		SvcResponseTopic: pResponseTopic,
	}

	return kc, nil
}
