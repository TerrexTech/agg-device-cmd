package device

import (
	"encoding/json"
	"log"

	"github.com/TerrexTech/go-eventstore-models/model"
	"github.com/TerrexTech/go-mongoutils/mongo"
	"github.com/TerrexTech/uuuid"
	"github.com/pkg/errors"
)

type deviceUpdate struct {
	Filter map[string]interface{} `json:"filter"`
	Update map[string]interface{} `json:"update"`
}

type updateResult struct {
	MatchedCount  int64 `json:"matchedCount,omitempty"`
	ModifiedCount int64 `json:"modifiedCount,omitempty"`
}

// Update handles "update" events.
func Update(collection *mongo.Collection, event *model.Event) *model.KafkaResponse {
	deviceUpdate := &deviceUpdate{}

	err := json.Unmarshal(event.Data, deviceUpdate)
	if err != nil {
		err = errors.Wrap(err, "Update: Error while unmarshalling Event-data")
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

	if len(deviceUpdate.Filter) == 0 {
		err = errors.New("blank filter provided")
		err = errors.Wrap(err, "Update")
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
	if len(deviceUpdate.Update) == 0 {
		err = errors.New("blank update provided")
		err = errors.Wrap(err, "Update")
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
	if deviceUpdate.Update["deviceID"] == (uuuid.UUID{}).String() {
		err = errors.New("found blank deviceID in update")
		err = errors.Wrap(err, "Update")
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

	updateStats, err := collection.UpdateMany(deviceUpdate.Filter, deviceUpdate.Update)
	if err != nil {
		err = errors.Wrap(err, "Update: Error in UpdateMany")
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

	result := &updateResult{
		MatchedCount:  updateStats.MatchedCount,
		ModifiedCount: updateStats.ModifiedCount,
	}
	resultMarshal, err := json.Marshal(result)
	if err != nil {
		err = errors.Wrap(err, "Update: Error marshalling Device Update-result")
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
		Result:        resultMarshal,
		EventAction:   event.EventAction,
		ServiceAction: event.ServiceAction,
		UUID:          event.UUID,
	}
}
