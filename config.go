package main

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	SetPoint         int32  `json:"setpoint,omitempty" bson:"setpoint,omitempty"`
	TempDifferential int32  `json:"tempdifferential,omitempty" bson:"tempdifferential,omitempty"`
	AParam           int32  `json:"aparam,omitempty" bson:"aparam,omitempty"`
	BParam           int32  `json:"bparam,omitempty" bson:"bparam,omitempty"`
	CParam           int32  `json:"cparam,omitempty" bson:"cparam,omitempty"`
	Scale            uint8  `json:"scale,omitempty" bson:"scale,omitempty"`
	Name             string `json:"name,omitempty" bson:"name,omitempty"`
	Id               string `json:"_id,omitempty" bson:"_id,omitempty"`
	Active           bool   `json:"active" bson:"active"`
}

type ConfigDB interface {
	CRUD[Config, string]
	GetCollection() *mongo.Collection
	Activate(string) error
}

type ConfigStore struct {
	mongodb *mongo.Database
}

func (store *ConfigStore) GetCollection() *mongo.Collection {
	return store.mongodb.Collection("config")
}

func (store *ConfigStore) Activate(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := store.GetCollection().UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"active": false}}); err != nil {
		return err
	}

	if _, err := store.GetCollection().UpdateByID(ctx, id, bson.M{"$set": bson.M{"active": true}}); err != nil {
		return err
	}

	return nil
}

func (store *ConfigStore) ReadOne(id string) (Config, error) {
	var res Config

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return res, err
	}

	if err := store.GetCollection().FindOne(ctx, bson.M{"_id": _id}).Decode(&res); err != nil {
		return res, err
	}

	return res, nil
}

func (store *ConfigStore) ReadMany() ([]Config, error) {
	var res []Config

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cur, err := store.GetCollection().
		Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1, "name": 1, "active": 1}))

	if err != nil {
		return res, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var conf Config

		if err := cur.Decode(&conf); err != nil {
			return res, err
		}

		res = append(res, conf)
	}

	return res, nil
}

func (store *ConfigStore) Create(conf Config) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var test Config

	// check for duplicates
	if err := store.GetCollection().FindOne(ctx, bson.M{"name": conf.Name}).Decode(&test); err == nil {
		return "", &DuplicateError{"config", fmt.Sprintf("document with %s already exists", conf.Name)}
	}
	// bsonDoc, err := bson.Marshal(conf)
	// if err != nil {
	// 	return "", err
	// }
	// insert document
	res, err := store.GetCollection().InsertOne(ctx, conf)
	if err != nil {
		return "", err
	}

	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (store *ConfigStore) Update(conf Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// update document
	if _, err := store.GetCollection().UpdateByID(ctx, conf.Id, bson.M{"$set": conf}); err != nil {
		return err
	}

	return nil
}

func (store *ConfigStore) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// delete document
	deleted, err := store.GetCollection().DeleteOne(ctx, bson.M{"_id": _id})
	if err != nil {
		return err
	}

	fmt.Printf("deleted %v docs\n", deleted.DeletedCount)

	return nil
}
