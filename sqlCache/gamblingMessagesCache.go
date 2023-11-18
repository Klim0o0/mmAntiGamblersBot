package sqlCache

import (
	"context"
	"github.com/jackc/pgx/v5"
	"log"
	"sync"
	"time"
)

type GamblingMessageInfo struct {
	UserChatIndicator
	MessageDate time.Time
	EmojiValue  int
	updated     bool
}

type UserChatIndicator struct {
	UserId int64
	ChatId int64
	Emoji  string
}

type GamblingMessageCache struct {
	cache        map[UserChatIndicator]*GamblingMessageInfo
	filedChats   map[int64]struct{}
	rwMutexChats sync.RWMutex
	conn         *pgx.Conn
	ctx          *context.Context
	rwMutex      sync.RWMutex
}

func CreateCache(conn *pgx.Conn, ctx context.Context) *GamblingMessageCache {
	cache := &GamblingMessageCache{cache: make(map[UserChatIndicator]*GamblingMessageInfo), conn: conn, ctx: &ctx, filedChats: make(map[int64]struct{})}
	go cache.startUpdateLoop()
	return cache
}

func (gamblingMessageCache *GamblingMessageCache) startUpdateLoop() {
	for {
		select {
		case <-time.After(120 * time.Second):
			err := gamblingMessageCache.insertCache()
			if err != nil {
				log.Println(err)
			}
		case <-(*gamblingMessageCache.ctx).Done():
			err := gamblingMessageCache.insertCache()
			if err != nil {
				log.Println(err)
			}
			return
		}

	}
}

func (gamblingMessageCache *GamblingMessageCache) Get(gamblingMessageInfo GamblingMessageInfo) (GamblingMessageInfo, bool) {
	gamblingMessageCache.fillChatCacheIfNeed(gamblingMessageInfo.UserChatIndicator, gamblingMessageInfo.MessageDate)

	gamblingMessageCache.rwMutex.RLock()
	msg, ok := gamblingMessageCache.cache[gamblingMessageInfo.UserChatIndicator]
	gamblingMessageCache.rwMutex.RUnlock()

	if ok && !msg.MessageDate.After(gamblingMessageInfo.MessageDate) {
		return *msg, true
	}
	return GamblingMessageInfo{}, false

}

func (gamblingMessageCache *GamblingMessageCache) Set(info GamblingMessageInfo) {
	gamblingMessageCache.rwMutex.Lock()
	gamblingMessageCache.cache[info.UserChatIndicator] = &info
	gamblingMessageCache.rwMutex.Unlock()
}

func (gamblingMessageCache *GamblingMessageCache) fillChatCacheIfNeed(gamblingMessageInfo UserChatIndicator, date time.Time) {
	gamblingMessageCache.rwMutexChats.Lock()
	if _, ok := gamblingMessageCache.filedChats[gamblingMessageInfo.ChatId]; !ok {
		gamblingMessageCache.fillChatCache(gamblingMessageInfo.ChatId, date)
		gamblingMessageCache.filedChats[gamblingMessageInfo.ChatId] = struct{}{}
	}
	gamblingMessageCache.rwMutexChats.Unlock()
}

func (gamblingMessageCache *GamblingMessageCache) insertCache() error {
	gamblingMessageCache.rwMutex.RLock()
	messagesToUpdate := make([]GamblingMessageInfo, 0, len(gamblingMessageCache.cache))

	for _, msg := range gamblingMessageCache.cache {
		if !msg.updated {
			msg.updated = true
			messagesToUpdate = append(messagesToUpdate, *msg)
		}
	}
	gamblingMessageCache.rwMutex.RUnlock()

	return insertMsg(gamblingMessageCache.conn, messagesToUpdate)
}

func (gamblingMessageCache *GamblingMessageCache) fillChatCache(id int64, v time.Time) {
	messages, _ := getLastChatMessages(gamblingMessageCache.conn, id, v)

	gamblingMessageCache.rwMutex.Lock()
	for i := range messages {
		gamblingMessageCache.cache[messages[i].UserChatIndicator] = &messages[i]
	}
	gamblingMessageCache.rwMutex.Unlock()
}

func getLastChatMessages(conn *pgx.Conn, id int64, v time.Time) ([]GamblingMessageInfo, error) {
	rows, err := conn.Query(context.Background(),
		"select * from tg_massages where masageDate > $1 and chatId=$2", v, id)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	gamblingMessageInfos, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (GamblingMessageInfo, error) {
		var gamblingMessageInfo GamblingMessageInfo
		err = rows.Scan(&gamblingMessageInfo.UserId, &gamblingMessageInfo.ChatId, &gamblingMessageInfo.MessageDate, &gamblingMessageInfo.Emoji, &gamblingMessageInfo.EmojiValue)
		gamblingMessageInfo.updated = true
		return gamblingMessageInfo, err
	})
	return gamblingMessageInfos, err
}

func insertMsg(conn *pgx.Conn, messages []GamblingMessageInfo) error {

	_, err := conn.CopyFrom(context.Background(), pgx.Identifier{"tg_massages"}, []string{"userid", "chatid", "masagedate", "emoji", "emoji_value"},
		pgx.CopyFromSlice(len(messages), func(i int) ([]any, error) {
			return []any{messages[i].UserId, messages[i].ChatId, messages[i].MessageDate, messages[i].Emoji, messages[i].EmojiValue}, nil
		}))

	return err
}

func GetAll(conn *pgx.Conn) ([]GamblingMessageInfo, bool) {

	query, err := conn.Query(context.Background(),
		"select * from tg_massages")
	defer query.Close()
	gamblingMessageInfos := make([]GamblingMessageInfo, 0, 0)
	for query.Next() {
		var gamblingMessageInfo GamblingMessageInfo
		err = query.Scan(&gamblingMessageInfo.UserId, &gamblingMessageInfo.ChatId, &gamblingMessageInfo.MessageDate, &gamblingMessageInfo.Emoji, &gamblingMessageInfo.EmojiValue)
		if err == nil {
			gamblingMessageInfo.updated = true
			gamblingMessageInfos = append(gamblingMessageInfos, gamblingMessageInfo)
		}
	}
	return gamblingMessageInfos, true
}

func (gamblingMessageCache *GamblingMessageCache) A(gamblingMessageInfo GamblingMessageInfo) bool {
	gamblingMessageCache.rwMutex.RLock()
	defer gamblingMessageCache.rwMutex.RUnlock()

	for k, _ := range gamblingMessageCache.cache {
		if k.ChatId == gamblingMessageInfo.ChatId {
			return false
		}
	}

	return false
}
