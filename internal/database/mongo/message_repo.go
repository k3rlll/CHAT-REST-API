package mongo_repo

import "go.mongodb.org/mongo-driver/mongo"

type MongoRepository struct {
	coll *mongo.Collection
}

func NewMongoRepository(db *mongo.Database, collectionName string) *MongoRepository {
	return &MongoRepository{
		coll: db.Collection(collectionName),
	}
}

func InitMongo(db *mongo.Database, collectionName string) *MongoRepository {}
