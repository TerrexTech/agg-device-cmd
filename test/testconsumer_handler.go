package test

import (
	"errors"
	"log"

	"github.com/Shopify/sarama"
)

// Handler for Consumer Messages
type msgHandler struct {
	msgCallback func(*sarama.ConsumerMessage) bool
}

func (*msgHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Println("Initializing Kafka MsgHandler")
	return nil
}

func (*msgHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Println("Closing Kafka MsgHandler")
	return nil
}

func (m *msgHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	if m.msgCallback == nil {
		return errors.New("msgCallback cannot be nil")
	}
	for msg := range claim.Messages() {
		session.MarkMessage(msg, "")

		val := m.msgCallback(msg)
		if val {
			return nil
		}
	}
	return errors.New("required value not found")
}
