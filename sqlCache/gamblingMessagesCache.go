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

func (gamblingMessageCache *GamblingMessageCache) A(gamblingMessageInfo GamblingMessageInfo) bool {
	gamblingMessageCache.rwMutex.RLock()

	for k, _ := range gamblingMessageCache.cache {
		if k.ChatId == gamblingMessageInfo.ChatId {
			gamblingMessageCache.rwMutex.RUnlock()
			return false
		}
	}

	gamblingMessageCache.rwMutex.RUnlock()
	return false
}

func (gamblingMessageCache *GamblingMessageCache) Get(gamblingMessageInfo GamblingMessageInfo) (GamblingMessageInfo, bool) {
	date := time.Date(gamblingMessageInfo.MessageDate.Year(), gamblingMessageInfo.MessageDate.Month(), gamblingMessageInfo.MessageDate.Day(), 0, 0, 0, 0, time.Local)

	gamblingMessageCache.rwMutexChats.Lock()
	if _, ok := gamblingMessageCache.filedChats[gamblingMessageInfo.ChatId]; !ok {
		gamblingMessageCache.FillChat(gamblingMessageInfo.ChatId, date)
		gamblingMessageCache.filedChats[gamblingMessageInfo.ChatId] = struct{}{}
	}
	gamblingMessageCache.rwMutexChats.Unlock()

	gamblingMessageCache.rwMutex.RLock()
	msg, ok := gamblingMessageCache.cache[gamblingMessageInfo.UserChatIndicator]
	gamblingMessageCache.rwMutex.RUnlock()
	if ok {
		if msg.MessageDate.Day() != gamblingMessageInfo.MessageDate.Day() {
			return GamblingMessageInfo{}, false
		}
		return *msg, true
	}

	newMsg, ok := gamblingMessageCache.getMsg(gamblingMessageInfo.UserChatIndicator, date)
	if ok {
		gamblingMessageCache.rwMutex.Lock()
		gamblingMessageCache.cache[gamblingMessageInfo.UserChatIndicator] = &newMsg
		gamblingMessageCache.rwMutex.Unlock()
	}
	return newMsg, ok
}
func (gamblingMessageCache *GamblingMessageCache) Set(info GamblingMessageInfo) {
	gamblingMessageCache.rwMutex.Lock()
	gamblingMessageCache.cache[info.UserChatIndicator] = &info
	gamblingMessageCache.rwMutex.Unlock()
}

func (gamblingMessageCache *GamblingMessageCache) startUpdateLoop() {
	for {
		select {
		case <-time.After(120 * time.Second):
			err := gamblingMessageCache.updateCache()
			if err != nil {
				log.Println(err)
			}
		case <-(*gamblingMessageCache.ctx).Done():
			err := gamblingMessageCache.updateCache()
			if err != nil {
				log.Println(err)
			}
			return
		}

	}
}

func (gamblingMessageCache *GamblingMessageCache) updateCache() error {
	gamblingMessageCache.rwMutex.RLock()
	messagesToUpdate := make([]GamblingMessageInfo, 0, len(gamblingMessageCache.cache))

	for _, msg := range gamblingMessageCache.cache {
		if !msg.updated {
			msg.updated = true
			messagesToUpdate = append(messagesToUpdate, *msg)
		}
	}
	gamblingMessageCache.rwMutex.RUnlock()

	return gamblingMessageCache.insertMsg(messagesToUpdate)
}

func (gamblingMessageCache *GamblingMessageCache) insertMsg(messages []GamblingMessageInfo) error {

	_, err := gamblingMessageCache.conn.CopyFrom(context.Background(), pgx.Identifier{"tg_massages"}, []string{"userid", "chatid", "masagedate", "emoji", "emoji_value"},
		pgx.CopyFromSlice(len(messages), func(i int) ([]any, error) {
			return []any{messages[i].UserId, messages[i].ChatId, messages[i].MessageDate, messages[i].Emoji, messages[i].EmojiValue}, nil
		}))

	return err
}

func (gamblingMessageCache *GamblingMessageCache) getMsg(indicator UserChatIndicator, date time.Time) (GamblingMessageInfo, bool) {
	query, err := gamblingMessageCache.conn.Query(context.Background(),
		"select * from tg_massages where userId=$1 and chatId=$2 and emoji = $3 and masageDate > $4",
		indicator.UserId, indicator.ChatId, indicator.Emoji, date)
	defer query.Close()
	var gamblingMessageInfo GamblingMessageInfo
	for query.Next() {
		err = query.Scan(&gamblingMessageInfo.UserId, &gamblingMessageInfo.ChatId, &gamblingMessageInfo.MessageDate, &gamblingMessageInfo.Emoji, &gamblingMessageInfo.EmojiValue)
		if err == nil {
			gamblingMessageInfo.updated = true
			return gamblingMessageInfo, true
		}
	}
	return gamblingMessageInfo, false
}

func (gamblingMessageCache *GamblingMessageCache) GetAll() ([]GamblingMessageInfo, bool) {

	query, err := gamblingMessageCache.conn.Query(context.Background(),
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

func (gamblingMessageCache *GamblingMessageCache) FillChat(id int64, v time.Time) {
	query, err := gamblingMessageCache.conn.Query(context.Background(),
		"select * from tg_massages where  masageDate > $1 and chatId=$2", v, id)
	defer query.Close()
	gamblingMessageInfos := make([]GamblingMessageInfo, 0, 100)
	for query.Next() {
		var gamblingMessageInfo GamblingMessageInfo
		err = query.Scan(&gamblingMessageInfo.UserId, &gamblingMessageInfo.ChatId, &gamblingMessageInfo.MessageDate, &gamblingMessageInfo.Emoji, &gamblingMessageInfo.EmojiValue)
		if err == nil {
			gamblingMessageInfo.updated = true
			gamblingMessageInfos = append(gamblingMessageInfos, gamblingMessageInfo)
		}
	}

	gamblingMessageCache.rwMutex.Lock()
	for i := range gamblingMessageInfos {
		gamblingMessageCache.cache[gamblingMessageInfos[i].UserChatIndicator] = &gamblingMessageInfos[i]
	}
	gamblingMessageCache.rwMutex.Unlock()
}
