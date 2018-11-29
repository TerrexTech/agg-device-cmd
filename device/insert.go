package device

import (
	"encoding/json"
	"log"

	"github.com/TerrexTech/go-eventstore-models/model"
	"github.com/TerrexTech/go-mongoutils/mongo"
	"github.com/TerrexTech/uuuid"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/pkg/errors"
)

// Insert handles "insert" events.
func Insert(collection *mongo.Collection, event *model.Event) *model.KafkaResponse {
	device := &Device{}
	err := json.Unmarshal(event.Data, device)
	if err != nil {
		err = errors.Wrap(err, "Insert: Error while unmarshalling Event-data")
		log.Println(err)
		return &model.KafkaResponse{
			AggregateID:   event.AggregateID,
			CorrelationID: event.CorrelationID,
			Error:         err.Error(),
			ErrorCode:     InternalError,
			EventAction:   event.EventAction,
			ServiceAction: event.ServiceAction,
			UUID:          event.UUID,
		}
	}

	if device.DeviceID == (uuuid.UUID{}) {
		err = errors.New("missing DeviceID")
		err = errors.Wrap(err, "Insert")
		log.Println(err)
		return &model.KafkaResponse{
			AggregateID:   event.AggregateID,
			CorrelationID: event.CorrelationID,
			Error:         err.Error(),
			ErrorCode:     InternalError,
			EventAction:   event.EventAction,
			ServiceAction: event.ServiceAction,
			UUID:          event.UUID,
		}
	}

	insertResult, err := collection.InsertOne(device)
	if err != nil {
		err = errors.Wrap(err, "Insert: Error Inserting Device into Mongo")
		log.Println(err)
		return &model.KafkaResponse{
			AggregateID:   event.AggregateID,
			CorrelationID: event.CorrelationID,
			Error:         err.Error(),
			ErrorCode:     DatabaseError,
			EventAction:   event.EventAction,
			ServiceAction: event.ServiceAction,
			UUID:          event.UUID,
		}
	}
	insertedID, assertOK := insertResult.InsertedID.(objectid.ObjectID)
	if !assertOK {
		err = errors.New("error asserting InsertedID from InsertResult to ObjectID")
		err = errors.Wrap(err, "Insert")
		log.Println(err)
		return &model.KafkaResponse{
			AggregateID:   event.AggregateID,
			CorrelationID: event.CorrelationID,
			Error:         err.Error(),
			ErrorCode:     InternalError,
			EventAction:   event.EventAction,
			ServiceAction: event.ServiceAction,
			UUID:          event.UUID,
		}
	}

	device.ID = insertedID
	result, err := json.Marshal(device)
	if err != nil {
		err = errors.Wrap(err, "Insert: Error marshalling Device Insert-result")
		log.Println(err)
		return &model.KafkaResponse{
			AggregateID:   event.AggregateID,
			CorrelationID: event.CorrelationID,
			Error:         err.Error(),
			ErrorCode:     InternalError,
			EventAction:   event.EventAction,
			ServiceAction: event.ServiceAction,
			UUID:          event.UUID,
		}
	}

	return &model.KafkaResponse{
		AggregateID:   event.AggregateID,
		CorrelationID: event.CorrelationID,
		Result:        result,
		EventAction:   event.EventAction,
		ServiceAction: event.ServiceAction,
		UUID:          event.UUID,
	}
}
