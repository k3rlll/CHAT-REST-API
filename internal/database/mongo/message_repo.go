package mongo_repo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionName = "messages"

type MongoRepository struct {
	coll *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		coll: db.Collection(collectionName),
	}
}

func InitMongo(ctx context.Context, uri, dbName string) (*mongo.Database, error) {
	clinet, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := clinet.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return clinet.Database(dbName), nil
}

func (r *MongoRepository) SaveMessage(ctx context.Context, msg interface{}) (string, error) {
	res, err := r.coll.InsertOne(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("failed to insert message: %w", err)
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("failed to convert inserted ID to ObjectID")
	}
	return id.Hex(), nil
}

func (r *MongoRepository) EditMessage(ctx context.Context, msg interface{})  {
