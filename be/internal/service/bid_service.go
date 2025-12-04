package service

import (
	"errors"
	"fmt"
	"log/slog"
	"milestone3/be/internal/entity"
	"milestone3/be/internal/repository"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	MinBidIncrement = 10000
	MaxRetries      = 3
)

var mutex sync.Map

func getMutex(itemID int64) *sync.Mutex {
	m, _ := mutex.LoadOrStore(itemID, &sync.Mutex{})
	return m.(*sync.Mutex)
}

func toLocalTime(t time.Time) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		t.Nanosecond(), time.Now().Location(),
	)
}

type bidService struct {
	redisRepo          repository.BidRedisRepository
	bidRepo            repository.BidRepository
	itemRepo           repository.AuctionItemRepository
	auctionSessionRepo repository.AuctionSessionRepository
	logger             *slog.Logger
}

type BidService interface {
	PlaceBid(sessionID, itemID, userID int64, amount float64, sessionEndTime time.Time) error
	GetHighestBid(sessionID, itemID int64) (float64, int64, error)

	SaveKeyToDB() error
	DeleteKeyValue() error
}

func NewBidService(r repository.BidRedisRepository, b repository.BidRepository, itemRepo repository.AuctionItemRepository, sessionRepo repository.AuctionSessionRepository, logger *slog.Logger) BidService {
	return &bidService{
		redisRepo:          r,
		bidRepo:            b,
		itemRepo:           itemRepo,
		auctionSessionRepo: sessionRepo,
		logger:             logger,
	}
}

func (s *bidService) PlaceBid(sessionID, itemID, userID int64, amount float64, sessionEndTime time.Time) error {
	if amount <= 0 {
		return ErrInvalidBidding
	}

	item, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		s.logger.Error("item not found", "itemID", itemID, "error", err)
		return ErrAuctionNotFound
	}

	if item.SessionID == nil || *item.SessionID != sessionID {
		return ErrInvalidAuction
	}

	if item.Status != "ongoing" {
		return ErrInvalidAuction
	}

	// validate session has started
	session, err := s.auctionSessionRepo.GetByID(sessionID)
	if err != nil {
		return ErrSessionNotFoundID
	}

	now := time.Now()

	sessionStart := toLocalTime(session.StartTime)
	sessionEnd := toLocalTime(session.EndTime)

	if now.Before(sessionStart) {
		return ErrInvalidAuction
	}

	if now.After(sessionEnd) {
		return ErrInvalidAuction
	}

	if err = s.redisRepo.CheckDuplicateBid(userID, itemID, amount, 10*time.Second); err != nil {
		return ErrDuplicateBid
	}

	// lock per item
	mu := getMutex(itemID)
	mu.Lock()
	defer mu.Unlock()

	currentHighest, currentBid, err := s.redisRepo.GetHighestBid(sessionID, itemID)
	if err != nil && !errors.Is(err, redis.Nil) {
		s.logger.Error("failed to get highest bid", "error", err)
		return err
	}

	// validate amount > starting price
	if currentHighest == 0 {
		if amount < item.StartingPrice {
			return ErrBidTooLow
		}
	}

	if currentHighest > 0 && amount <= currentHighest {
		return ErrBidTooLow
	}

	if currentBid == userID {
		return ErrAlreadyHighestBidder
	}

	if currentHighest > 0 && amount < currentHighest+MinBidIncrement {
		return ErrBidTooLow
	}

	if err := s.redisRepo.SetHighestBid(sessionID, itemID, amount, userID, sessionEndTime); err != nil {
		s.logger.Error("failed to set highest bid", "error", err)
		return err
	}

	s.logger.Info("new highest bid",
		"sessionID", sessionID,
		"itemID", itemID,
		"userID", userID,
		"oldBid", currentHighest,
		"newBid", amount,
	)

	return nil
}

func (s *bidService) GetHighestBid(sessionID, itemID int64) (float64, int64, error) {
	_, err := s.itemRepo.GetByID(itemID)
	if err != nil {
		return 0, 0, ErrAuctionNotFound
	}
	return s.redisRepo.GetHighestBid(sessionID, itemID)
}

func parseKey(key string) (sessionID, itemID int64, err error) {
	parts := strings.Split(key, ":")

	if len(parts) != 5 {
		return 0, 0, errors.New("invalid key format")
	}

	if parts[0] != "active" || parts[1] != "auction" || parts[3] != "item" {
		return 0, 0, errors.New("invalid key format, expected: active:auction:{sessionID}:item:{itemID}")
	}

	sessionID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid sessionID: %w", err)
	}

	itemID, err = strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid itemID: %w", err)
	}

	return sessionID, itemID, nil
}

func (s *bidService) SaveKeyToDB() error {
	pattern := "active:auction:*:item:*"

	keys, err := s.redisRepo.ScanKeys(pattern)
	if err != nil {
		s.logger.Error("failed to scan keys", "pattern", pattern, "error", err)
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	s.logger.Info("processing expired bids", "totalKeys", len(keys))

	totalSavedAuctionItemToDB := 0
	for _, key := range keys {
		parsedSessionID, itemID, err := parseKey(key)
		if err != nil {
			s.logger.Warn("invalid key format", "key", key, "error", err)
			continue
		}

		// fetch end_time from database for accuracy
		session, err := s.auctionSessionRepo.GetByID(parsedSessionID)
		if err != nil {
			s.logger.Warn("failed to get session from DB", "key", key, "sessionID", parsedSessionID, "error", err)
			continue
		}

		now := time.Now()
		endTimeLocal := toLocalTime(session.EndTime)

		if now.Before(endTimeLocal) {
			continue
		}

		// get bid data from redis
		bid, err := s.redisRepo.GetBidByKey(key)
		if err != nil {
			s.logger.Warn("failed to get bid", "key", key, "error", err)
			continue
		}

		// save final bid to DB
		err = s.bidRepo.SaveFinalBid(&entity.Bid{
			ItemID: itemID,
			UserID: bid.UserID,
			Amount: bid.Amount,
		})
		if err != nil {
			s.logger.Error("failed to save final bid",
				"sessionID", parsedSessionID,
				"itemID", itemID,
				"error", err,
			)
			continue
		}

		// set status to finished
		item, err := s.itemRepo.GetByID(itemID)
		if err != nil {
			s.logger.Warn("failed to get item for status update", "itemID", itemID, "error", err)
		} else {
			item.Status = "finished"
			if err := s.itemRepo.Update(item); err != nil {
				s.logger.Error("failed to update item status to finished", "itemID", itemID, "error", err)
			} else {
				s.logger.Info("item status updated to finished", "itemID", itemID)
			}
		}

		// delete redis key after saved to table Bid
		if err := s.redisRepo.DeleteKey(key); err != nil {
			s.logger.Warn("failed to delete Redis key after save", "key", key, "error", err)
		}

		s.logger.Info("final bid saved",
			"sessionID", parsedSessionID,
			"itemID", itemID,
			"amount", bid.Amount,
			"winner", bid.UserID,
		)
		totalSavedAuctionItemToDB++
	}

	if totalSavedAuctionItemToDB > 0 {
		s.logger.Info("expired sessions processed", "totalSaved", totalSavedAuctionItemToDB)
	}
	return nil
}

func (s *bidService) DeleteKeyValue() error {
	pattern := "active:auction:*:item:*"
	keys, err := s.redisRepo.ScanKeys(pattern)
	if err != nil {
		s.logger.Error("failed to scan keys for cleanup", "pattern", pattern, "error", err)
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	deletedCount := 0
	for _, key := range keys {
		if err := s.redisRepo.DeleteKey(key); err != nil {
			s.logger.Warn("failed to delete key during cleanup", "key", key, "error", err)
		} else {
			deletedCount++
		}
	}

	s.logger.Info("midnight cleanup completed", "deleted", deletedCount)
	return nil
}
