package mongo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	dom "main/internal/domain/entity"
	"main/pkg/customerrors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionName = "messages"

var (
	errMessageNotFound = errors.New("message not found")
	errMongoDB         = errors.New("mongo database error")
)

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

func (r *MessageRepository) EditMessage(ctx context.Context, senderID int64, chatID int64, msgID string, newText string) (int64, error) {
	objID, err := primitive.ObjectIDFromHex(msgID)
	if err != nil {
		return 0, err
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
		return 0, fmt.Errorf("failed to update message: %w", err)
	}

	return res.MatchedCount, nil
}

func (r *MessageRepository) DeleteMessage(ctx context.Context, senderID, chatID int64, msgID []string) (int64, error) {

	oids := make([]primitive.ObjectID, 0, len(msgID))
	for _, id := range msgID {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return 0, fmt.Errorf("invalid message ID: %w", err)
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

func (r *MessageRepository) GetMessages(ctx context.Context, chatID int64, anchorTime time.Time, anchorID string, limit int64) ([]dom.Message, error) {
	findOptions := options.Find().
		SetLimit(limit).
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}})

	filter := bson.M{"chat_id": chatID}

	if !anchorTime.IsZero() {
		var objID primitive.ObjectID
		if anchorID != "" {
			objID, _ = primitive.ObjectIDFromHex(anchorID)
		}

		filter["$or"] = []bson.M{
			{"created_at": bson.M{"$lt": anchorTime}},
			{
				"created_at": anchorTime,
				"_id":        bson.M{"$lt": objID},
			},
		}
	}

	cursor, err := r.coll.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("mongo find error: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []dom.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	if messages == nil {
		messages = []dom.Message{}
	}

	return messages, nil
}

func (r *MessageRepository) GetLatestMessage(ctx context.Context, chatID int64) (dom.Message, error) {
	filter := bson.M{"chat_id": chatID}

	opts := options.FindOne().SetSort(bson.D{
		{Key: "created_at", Value: -1},
		{Key: "_id", Value: -1},
	})

	var msg dom.Message
	err := r.coll.FindOne(ctx, filter, opts).Decode(&msg)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return dom.Message{}, customerrors.ErrMessageDoesNotExists
		}
		return dom.Message{}, fmt.Errorf("failed to get latest message: %w", err)
	}

	return msg, nil
}
