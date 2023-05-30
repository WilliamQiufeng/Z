package sessions

import (
	"example.com/Quaver/Z/common"
	"example.com/Quaver/Z/db"
	"example.com/Quaver/Z/objects"
	"example.com/Quaver/Z/utils"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type User struct {
	// The connection for the user
	Conn net.Conn

	// The token used to identify the user for requests.
	token string

	// All user table information from the database
	Info *db.User

	// mutex for all operations regarding changes in the user
	mutex *sync.Mutex

	// Player statistics from the database
	stats map[common.Mode]*db.UserStats

	// The last time the user was pinged
	lastPingTimestamp int64

	// The last time the user sent a successful pong
	lastPongTimestamp int64

	// The last detected processes that were discovered on the user
	lastDetectedProcesses []string

	// The current client status of the user
	status *objects.ClientStatus
}

type PacketUser struct {
	Id          int               `json:"id"`
	SteamId     string            `json:"sid"`
	Username    string            `json:"u"`
	UserGroups  common.UserGroups `json:"ug"`
	MuteEndTime int64             `json:"m"`
	Country     string            `json:"c"`
}

// NewUser Creates a new user session struct object
func NewUser(conn net.Conn, user *db.User) *User {
	return &User{
		Conn:              conn,
		token:             utils.GenerateRandomString(64),
		Info:              user,
		mutex:             &sync.Mutex{},
		stats:             map[common.Mode]*db.UserStats{},
		lastPingTimestamp: time.Now().UnixMilli(),
		lastPongTimestamp: time.Now().UnixMilli(),
		status: &objects.ClientStatus{
			Status:    0,
			MapId:     -1,
			MapMd5:    "",
			GameMode:  common.ModeKeys4,
			Content:   "",
			Modifiers: 0,
		},
	}
}

// GetToken Returns the user token
func (u *User) GetToken() string {
	return u.token
}

// GetStats Retrieves the stats for the user
func (u *User) GetStats() map[common.Mode]*db.UserStats {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.stats
}

// SetStats Updates the statistics for the user
func (u *User) SetStats() error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	for i := 1; i < int(common.ModeEnumMaxValue); i++ {
		mode := common.Mode(i)
		stats, err := db.GetUserStats(u.Info.Id, u.Info.Country, mode)

		if err != nil {
			return err
		}

		u.stats[mode] = stats
	}

	return nil
}

// GetLastPingTimestamp Retrieves the last ping timestamp
func (u *User) GetLastPingTimestamp() int64 {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.lastPingTimestamp
}

// SetLastPingTimestamp Sets the last ping timestamp to the current time
func (u *User) SetLastPingTimestamp() {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.lastPingTimestamp = time.Now().UnixMilli()
}

// GetLastPongTimestamp Retrieves the last pong timestamp
func (u *User) GetLastPongTimestamp() int64 {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.lastPongTimestamp
}

// SetLastPongTimestamp Sets the last pong timestamp to the current time
func (u *User) SetLastPongTimestamp() {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.lastPongTimestamp = time.Now().UnixMilli()
}

// GetLastDetectedProcesses Gets the last detected processes for the user
func (u *User) GetLastDetectedProcesses() []string {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.lastDetectedProcesses
}

// SetLastDetectedProcesses Sets the last detected processes for the user
func (u *User) SetLastDetectedProcesses(processes []string) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.lastDetectedProcesses = processes
}

// GetClientStatus Gets the current user client status
func (u *User) GetClientStatus() *objects.ClientStatus {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.status
}

// SetClientStatus Sets the current user client status
func (u *User) SetClientStatus(status *objects.ClientStatus) {
	u.mutex.Lock()
	u.status = status
	u.mutex.Unlock()

	err := addUserClientStatusToRedis(u)

	if err != nil {
		log.Println(err)
	}
}

// IsMuted Returns if the user is muted
func (u *User) IsMuted() bool {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return u.Info.MuteEndTime > time.Now().UnixMilli()
}

// SerializeForPacket Serializes the user to be used in a packet
func (u *User) SerializeForPacket() *PacketUser {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	return &PacketUser{
		Id:          u.Info.Id,
		SteamId:     u.Info.SteamId,
		Username:    u.Info.Username,
		UserGroups:  u.Info.UserGroups,
		MuteEndTime: u.Info.MuteEndTime,
		Country:     u.Info.Country,
	}
}

// Returns the Redis key for the user's session
func (u *User) getRedisSessionKey() string {
	return fmt.Sprintf("quaver:server:session:%v", u.token)
}

// Returns the Redis key for the user's client
func (u *User) getRedisClientStatusKey() string {
	return fmt.Sprintf("quaver:server:user_status:%v", u.Info.Id)
}
