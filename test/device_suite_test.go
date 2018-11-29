package test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/TerrexTech/agg-device-cmd/device"
	"github.com/TerrexTech/go-commonutils/commonutil"
	"github.com/TerrexTech/go-eventstore-models/model"
	"github.com/TerrexTech/go-kafkautils/kafka"
	"github.com/TerrexTech/uuuid"
	"github.com/joho/godotenv"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

func Byf(s string, args ...interface{}) {
	By(fmt.Sprintf(s, args...))
}

func TestDevice(t *testing.T) {
	log.Println("Reading environment file")
	err := godotenv.Load("../.env")
	if err != nil {
		err = errors.Wrap(err,
			".env file not found, env-vars will be read as set in environment",
		)
		log.Println(err)
	}

	missingVar, err := commonutil.ValidateEnv(
		"KAFKA_BROKERS",
		"KAFKA_CONSUMER_EVENT_GROUP",

		"KAFKA_CONSUMER_EVENT_TOPIC",
		"KAFKA_CONSUMER_EVENT_QUERY_GROUP",
		"KAFKA_CONSUMER_EVENT_QUERY_TOPIC",

		"KAFKA_PRODUCER_EVENT_TOPIC",
		"KAFKA_PRODUCER_EVENT_QUERY_TOPIC",
		"KAFKA_PRODUCER_RESPONSE_TOPIC",

		"MONGO_HOSTS",
		"MONGO_USERNAME",
		"MONGO_PASSWORD",
		"MONGO_DATABASE",
		"MONGO_CONNECTION_TIMEOUT_MS",
		"MONGO_RESOURCE_TIMEOUT_MS",
	)

	if err != nil {
		err = errors.Wrapf(err, "Env-var %s is required for testing, but is not set", missingVar)
		log.Fatalln(err)
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "DeviceAggregate Suite")
}

var _ = Describe("DeviceAggregate", func() {
	var (
		kafkaBrokers          []string
		eventsTopic           string
		producerResponseTopic string

		mockDevice *device.Device
		mockEvent  *model.Event
	)
	BeforeSuite(func() {
		kafkaBrokers = *commonutil.ParseHosts(
			os.Getenv("KAFKA_BROKERS"),
		)
		eventsTopic = os.Getenv("KAFKA_PRODUCER_EVENT_TOPIC")
		producerResponseTopic = os.Getenv("KAFKA_PRODUCER_RESPONSE_TOPIC")

		deviceID, err := uuuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		mockDevice = &device.Device{
			DeviceID:      deviceID,
			DateInstalled: time.Now().Unix(),
			Lot:           "test-lot",
			Status:        "ok",
		}
		marshalDevice, err := json.Marshal(mockDevice)
		Expect(err).ToNot(HaveOccurred())

		log.Println(string(marshalDevice))
		cid, err := uuuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		uid, err := uuuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		uuid, err := uuuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		mockEvent = &model.Event{
			EventAction:   "insert",
			CorrelationID: cid,
			AggregateID:   device.AggregateID,
			Data:          marshalDevice,
			NanoTime:      time.Now().UnixNano(),
			UserUUID:      uid,
			UUID:          uuid,
			Version:       0,
			YearBucket:    2018,
		}
	})

	Describe("Device Operations", func() {
		It("should insert record", func(done Done) {
			Byf("Producing MockEvent")
			p, err := kafka.NewProducer(&kafka.ProducerConfig{
				KafkaBrokers: kafkaBrokers,
			})
			Expect(err).ToNot(HaveOccurred())
			marshalEvent, err := json.Marshal(mockEvent)
			Expect(err).ToNot(HaveOccurred())
			p.Input() <- kafka.CreateMessage(eventsTopic, marshalEvent)

			// Check if MockEvent was processed correctly
			Byf("Consuming Result")
			c, err := kafka.NewConsumer(&kafka.ConsumerConfig{
				KafkaBrokers: kafkaBrokers,
				GroupName:    "aggDevice.test.group.1",
				Topics:       []string{producerResponseTopic},
			})
			msgCallback := func(msg *sarama.ConsumerMessage) bool {
				defer GinkgoRecover()
				kr := &model.KafkaResponse{}
				err := json.Unmarshal(msg.Value, kr)
				Expect(err).ToNot(HaveOccurred())

				if kr.UUID == mockEvent.UUID {
					Expect(kr.Error).To(BeEmpty())
					Expect(kr.ErrorCode).To(BeZero())
					Expect(kr.CorrelationID).To(Equal(mockEvent.CorrelationID))
					Expect(kr.UUID).To(Equal(mockEvent.UUID))

					device := &device.Device{}
					err = json.Unmarshal(kr.Result, device)
					Expect(err).ToNot(HaveOccurred())

					log.Printf("%+v", mockDevice.DeviceID)
					if device.DeviceID == mockDevice.DeviceID {
						mockDevice.ID = device.ID
						Expect(device).To(Equal(mockDevice))
						return true
					}
				}
				return false
			}

			handler := &msgHandler{msgCallback}
			c.Consume(context.Background(), handler)

			Byf("Checking if record got inserted into Database")
			aggColl, err := loadAggCollection()
			Expect(err).ToNot(HaveOccurred())
			findResult, err := aggColl.FindOne(mockDevice)
			Expect(err).ToNot(HaveOccurred())
			findDevice, assertOK := findResult.(*device.Device)
			Expect(assertOK).To(BeTrue())
			Expect(findDevice).To(Equal(mockDevice))
			log.Fatalln("")
			close(done)
		}, 2000)

		It("should update record", func(done Done) {
			Byf("Creating update args")
			filterDevice := map[string]interface{}{
				"deviceID": mockDevice.DeviceID,
			}
			mockDevice.Status = "errored"
			mockDevice.Lot = "new-lot"
			// Remove ObjectID because this is not passed from gateway
			mockID := mockDevice.ID
			mockDevice.ID = objectid.NilObjectID
			update := map[string]interface{}{
				"filter": filterDevice,
				"update": mockDevice,
			}
			marshalUpdate, err := json.Marshal(update)
			Expect(err).ToNot(HaveOccurred())
			// Reassign back ID so we can compare easily with database-entry
			mockDevice.ID = mockID

			Byf("Creating update MockEvent")
			uuid, err := uuuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			mockEvent.EventAction = "update"
			mockEvent.Data = marshalUpdate
			mockEvent.NanoTime = time.Now().UnixNano()
			mockEvent.UUID = uuid

			Byf("Producing MockEvent")
			p, err := kafka.NewProducer(&kafka.ProducerConfig{
				KafkaBrokers: kafkaBrokers,
			})
			Expect(err).ToNot(HaveOccurred())
			marshalEvent, err := json.Marshal(mockEvent)
			Expect(err).ToNot(HaveOccurred())
			p.Input() <- kafka.CreateMessage(eventsTopic, marshalEvent)

			// Check if MockEvent was processed correctly
			Byf("Consuming Result")
			c, err := kafka.NewConsumer(&kafka.ConsumerConfig{
				KafkaBrokers: kafkaBrokers,
				GroupName:    "aggdevice.test.group.1",
				Topics:       []string{producerResponseTopic},
			})
			msgCallback := func(msg *sarama.ConsumerMessage) bool {
				defer GinkgoRecover()
				kr := &model.KafkaResponse{}
				err := json.Unmarshal(msg.Value, kr)
				Expect(err).ToNot(HaveOccurred())

				if kr.UUID == mockEvent.UUID {
					Expect(kr.Error).To(BeEmpty())
					Expect(kr.ErrorCode).To(BeZero())
					Expect(kr.CorrelationID).To(Equal(mockEvent.CorrelationID))
					Expect(kr.UUID).To(Equal(mockEvent.UUID))

					result := map[string]int{}
					err = json.Unmarshal(kr.Result, &result)
					Expect(err).ToNot(HaveOccurred())

					if result["matchedCount"] != 0 && result["modifiedCount"] != 0 {
						Expect(result["matchedCount"]).To(Equal(1))
						Expect(result["modifiedCount"]).To(Equal(1))
						return true
					}
				}
				return false
			}

			handler := &msgHandler{msgCallback}
			c.Consume(context.Background(), handler)

			Byf("Checking if record got inserted into Database")
			aggColl, err := loadAggCollection()
			Expect(err).ToNot(HaveOccurred())
			findResult, err := aggColl.FindOne(mockDevice)
			Expect(err).ToNot(HaveOccurred())
			findDevice, assertOK := findResult.(*device.Device)
			Expect(assertOK).To(BeTrue())
			Expect(findDevice).To(Equal(mockDevice))

			close(done)
		}, 20)

		It("should delete record", func(done Done) {
			Byf("Creating delete args")
			deleteArgs := map[string]interface{}{
				"deviceID": mockDevice.DeviceID,
			}
			marshalDelete, err := json.Marshal(deleteArgs)
			Expect(err).ToNot(HaveOccurred())

			Byf("Creating delete MockEvent")
			uuid, err := uuuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			mockEvent.EventAction = "delete"
			mockEvent.Data = marshalDelete
			mockEvent.NanoTime = time.Now().UnixNano()
			mockEvent.UUID = uuid

			Byf("Producing MockEvent")
			p, err := kafka.NewProducer(&kafka.ProducerConfig{
				KafkaBrokers: kafkaBrokers,
			})
			Expect(err).ToNot(HaveOccurred())
			marshalEvent, err := json.Marshal(mockEvent)
			Expect(err).ToNot(HaveOccurred())
			p.Input() <- kafka.CreateMessage(eventsTopic, marshalEvent)

			// Check if MockEvent was processed correctly
			Byf("Consuming Result")
			c, err := kafka.NewConsumer(&kafka.ConsumerConfig{
				KafkaBrokers: kafkaBrokers,
				GroupName:    "aggdevice.test.group.1",
				Topics:       []string{producerResponseTopic},
			})
			msgCallback := func(msg *sarama.ConsumerMessage) bool {
				defer GinkgoRecover()
				kr := &model.KafkaResponse{}
				err := json.Unmarshal(msg.Value, kr)
				Expect(err).ToNot(HaveOccurred())

				if kr.UUID == mockEvent.UUID {
					Expect(kr.Error).To(BeEmpty())
					Expect(kr.ErrorCode).To(BeZero())
					Expect(kr.CorrelationID).To(Equal(mockEvent.CorrelationID))
					Expect(kr.UUID).To(Equal(mockEvent.UUID))

					result := map[string]int{}
					err = json.Unmarshal(kr.Result, &result)
					Expect(err).ToNot(HaveOccurred())

					if result["deletedCount"] != 0 {
						Expect(result["deletedCount"]).To(Equal(1))
						return true
					}
				}
				return false
			}

			handler := &msgHandler{msgCallback}
			c.Consume(context.Background(), handler)

			Byf("Checking if record got inserted into Database")
			aggColl, err := loadAggCollection()
			Expect(err).ToNot(HaveOccurred())
			_, err = aggColl.FindOne(mockDevice)
			Expect(err).To(HaveOccurred())

			close(done)
		}, 20)
	})
})
