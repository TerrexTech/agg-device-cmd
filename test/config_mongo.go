package test

import (
	"os"

	"github.com/TerrexTech/agg-device-cmd/device"

	"github.com/TerrexTech/go-commonutils/commonutil"
	"github.com/TerrexTech/go-mongoutils/mongo"
)

func loadAggCollection() (*mongo.Collection, error) {
	database := os.Getenv("MONGO_DATABASE")
	aggCollection := os.Getenv("MONGO_AGG_COLLECTION")

	return loadMongoCollection(database, aggCollection, &device.Device{})
}

func loadMongoCollection(
	db string, collection string, schemaStruct interface{},
) (*mongo.Collection, error) {
	hosts := *commonutil.ParseHosts(
		os.Getenv("MONGO_HOSTS"),
	)
	username := os.Getenv("MONGO_USERNAME")
	password := os.Getenv("MONGO_PASSWORD")

	mongoConfig := mongo.ClientConfig{
		Hosts:               hosts,
		Username:            username,
		Password:            password,
		TimeoutMilliseconds: 5000,
	}

	client, err := mongo.NewClient(mongoConfig)
	if err != nil {
		return nil, err
	}

	conn := &mongo.ConnectionConfig{
		Client:  client,
		Timeout: 5000,
	}
	c := &mongo.Collection{
		Connection:   conn,
		Database:     db,
		Name:         collection,
		SchemaStruct: schemaStruct,
	}
	coll, err := mongo.EnsureCollection(c)
	if err != nil {
		return nil, err
	}
	return coll, nil
}
