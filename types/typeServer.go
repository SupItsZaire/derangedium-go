package types

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/supitszaire/derangedium-go/global"
)

var Servers = make(map[string]*Context)

type Context struct {
	OptedInUsers        map[string]time.Time    // userIDs -> opt-in timestamp
	WhitelistedChannels map[string]bool         // channelIDs
	GuildModel          *MarkovChain            // server-wide model
	ChannelModels       map[string]*MarkovChain // channelID -> model
	Mu                  sync.RWMutex
}

func GetServerData(guildID string) *Context {
	global.DataLock.Lock()
	defer global.DataLock.Unlock()

	if Servers[guildID] == nil {
		Servers[guildID] = &Context{
			OptedInUsers:        make(map[string]time.Time),
			WhitelistedChannels: make(map[string]bool),
			GuildModel:          NewMarkovChain(),
			ChannelModels:       make(map[string]*MarkovChain),
		}
	}
	return Servers[guildID]
}

// json version for data.json
type contextJSON struct {
	OptedInUsers        map[string]string              `json:"opted_in_users"`
	WhitelistedChannels map[string]bool                `json:"whitelisted_channels"`
	GuildModel          map[string][]string            `json:"guild_model"`
	ChannelModels       map[string]map[string][]string `json:"channel_models"`
}

func SaveData(filename string) error {
	global.DataLock.RLock()
	defer global.DataLock.RUnlock()

	data := make(map[string]contextJSON)

	for guildID, ctx := range Servers {
		ctx.Mu.RLock()

		// timestamp to string
		optedInUsers := make(map[string]string)
		for userID, timestamp := range ctx.OptedInUsers {
			optedInUsers[userID] = timestamp.Format(time.RFC3339)
		}

		// model to json conversion
		channelModels := make(map[string]map[string][]string)
		for chanID, model := range ctx.ChannelModels {
			model.Mu.RLock()
			channelModels[chanID] = model.Chain
			model.Mu.RUnlock()
		}

		ctx.GuildModel.Mu.RLock()
		data[guildID] = contextJSON{
			OptedInUsers:        optedInUsers,
			WhitelistedChannels: ctx.WhitelistedChannels,
			GuildModel:          ctx.GuildModel.Chain,
			ChannelModels:       channelModels,
		}
		ctx.GuildModel.Mu.RUnlock()
		ctx.Mu.RUnlock()
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func LoadData(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("No pre-existing data file present, starting anew")
			return nil
		}
		return err
	}
	defer file.Close()

	var data map[string]contextJSON
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	global.DataLock.Lock()
	defer global.DataLock.Unlock()

	for guildID, ctxJSON := range data {

		optedInUsers := make(map[string]time.Time)
		for userID, timestampStr := range ctxJSON.OptedInUsers {
			timestamp, err := time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				timestamp = time.Time{}
			}
			optedInUsers[userID] = timestamp
		}

		ctx := &Context{
			OptedInUsers:        optedInUsers,
			WhitelistedChannels: ctxJSON.WhitelistedChannels,
			GuildModel: &MarkovChain{
				Chain: ctxJSON.GuildModel,
			},
			ChannelModels: make(map[string]*MarkovChain),
		}

		for chanID, chain := range ctxJSON.ChannelModels {
			ctx.ChannelModels[chanID] = &MarkovChain{
				Chain: chain,
			}
		}

		Servers[guildID] = ctx
	}

	log.Printf("Loaded data for %d servers", len(data))
	return nil
}
