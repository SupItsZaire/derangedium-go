package types

import (
	"sync"

	"github.com/supitszaire/derangedium-go/global"
)

var Servers = make(map[string]*Context)

type Context struct {
	OptedInUsers        map[string]bool         // userIDs
	WhitelistedChannels map[string]bool         // channelIDs
	GuildModel          *MarkovChain            // server-wide -> model
	ChannelModels       map[string]*MarkovChain // channelID -> model
	mu                  sync.RWMutex
}

func getServerData(guildID string) *Context {
	global.DataLock.Lock()
	defer global.DataLock.Unlock()

	if Servers[guildID] == nil {
		Servers[guildID] = &Context{
			OptedInUsers:        make(map[string]bool),
			WhitelistedChannels: make(map[string]bool),
			GuildModel:          NewMarkovChain(),
			ChannelModels:       make(map[string]*MarkovChain),
		}
	}
	return Servers[guildID]
}
