package mongo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	dom "main/internal/domain/entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	errMessageNotFound = errors.New("message not found")
	errMongoDB         = errors.New("mongo database error")
)

const collectionName = "messages"

func NewMessageRepository(db *mongo.Database, logger *slog.Logger) *MessageRepository {
	collection := db.Collection("messages")
	repo := &MessageRepository{
		coll: collection,
	}

	if err := repo.initIndexes(); err != nil {
		logger.Warn("Indexes did not start")
		return nil

	}

	return repo
}

type MessageRepository struct {
	logger *slog.Logger
	coll   *mongo.Collection
}

func (r *MessageRepository) initIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indlexModel := []mongo.IndexModel{{
		Keys: bson.D{
			{Key: "chat_id", Value: 1},
			{Key: "created_at", Value: -1},
		},
	},
		{
			Keys: bson.D{
				{Key: "sender_id", Value: 1},
			},
		},
	}
	_, err := r.coll.Indexes().CreateMany(ctx, indlexModel)
	if err != nil {
		return err
	}
	return nil
}

func (r *MessageRepository) SaveMessage(ctx context.Context, msg interface{}) (string, error) {
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

func (r *MessageRepository) EditMessage(ctx context.Context, senderID int64, chatID int64, msgID string, newText string) error {
	objID, err := primitive.ObjectIDFromHex(msgID)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id":       objID,
		"sender_id": senderID,
		"chat_id":   chatID,
	}

	update := bson.M{
		"$set": bson.M{
			"text":       newText,
			"updated_at": time.Now(),
		},
	}

	res, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	if res.MatchedCount == 0 {
		return errMongoDB
	}
	return nil
}

func (r *MessageRepository) DeleteMessage(ctx context.Context, senderID, chatID int64, msgID []string) (int64, error) {

	oids := make([]primitive.ObjectID, 0, len(msgID))
	for _, id := range msgID {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			r.logger.Warn("invalid message ID", "id", id, "error", err)
			continue
		}
		oids = append(oids, objID)
	}

	filter := bson.M{
		"_id":       bson.M{"$in": oids},
		"sender_id": senderID,
		"chat_id":   chatID,
	}

	res, err := r.coll.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete messages: %w", err)
	}

	return res.DeletedCount, nil
}

func (r *MessageRepository) GetMessages(ctx context.Context, chatID, limit int64, lastMessage time.Time) ([]dom.Message, error) {
	findOptions := options.Find()

	findOptions.SetLimit(limit)

	findOptions.SetSort(bson.D{{Key: "created_at", Value: -1}})

	filter := bson.M{
		"chat_id":    chatID,
		"created_at": bson.M{"$lt": lastMessage},
	}

	cursor, err := r.coll.Find(ctx, filter, findOptions)
	if err != nil {
		return []dom.Message{}, fmt.Errorf("failed to find: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []dom.Message

	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}
	return messages, nil

}
