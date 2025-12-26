package handlers

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/supitszaire/derangedium-go/types"
)

func cleanMessage(content string, s *discordgo.Session) string {
	content = strings.TrimSpace(content)
	for {
		start := strings.Index(content, "<@")
		if start == -1 {
			break
		}
		end := strings.Index(content[start:], ">")
		if end == -1 {
			break
		}
		content = content[:start] + content[start+end+1:]
	}
	for {
		start := strings.Index(content, "<@&")
		if start == -1 {
			break
		}
		end := strings.Index(content[start:], ">")
		if end == -1 {
			break
		}
		content = content[:start] + content[start+end+1:]
	}
	return strings.TrimSpace(content)
}

func Pretrain(s *discordgo.Session, guildID, channelID string) int {
	ctx := types.GetServerData(guildID)

	var lastID string
	totalMessages := 0

	for {
		messages, err := s.ChannelMessages(channelID, 100, lastID, "", "")
		if err != nil || len(messages) == 0 {
			break
		}

		for _, msg := range messages {
			ctx.Mu.RLock()
			optInTime, optedIn := ctx.OptedInUsers[msg.Author.ID]
			ctx.Mu.RUnlock()

			if !optedIn {
				continue
			}

			msgTime := msg.Timestamp

			if !optInTime.IsZero() && msgTime.Before(optInTime) {
				continue
			}

			content := cleanMessage(msg.Content, s)
			if content != "" && !strings.HasPrefix(content, "/") {
				ctx.GuildModel.Train(content)

				ctx.Mu.Lock()
				if ctx.ChannelModels[channelID] != nil {
					ctx.ChannelModels[channelID].Train(content)
				}
				ctx.Mu.Unlock()

				totalMessages++
			}
		}

		lastID = messages[len(messages)-1].ID
		time.Sleep(time.Second) // Rate limit
	}

	return totalMessages
}

func PretrainUser(s *discordgo.Session, guildID, userID string, since time.Time) int {
	ctx := types.GetServerData(guildID)
	totalMessages := 0

	ctx.Mu.RLock()
	channels := make([]string, 0, len(ctx.WhitelistedChannels))
	for chanID := range ctx.WhitelistedChannels {
		channels = append(channels, chanID)
	}
	ctx.Mu.RUnlock()

	for _, channelID := range channels {
		var lastID string

		for {
			messages, err := s.ChannelMessages(channelID, 100, lastID, "", "")
			if err != nil || len(messages) == 0 {
				break
			}

			for _, msg := range messages {
				if msg.Author.ID != userID {
					continue
				}

				msgTime := msg.Timestamp

				if !since.IsZero() && msgTime.Before(since) {
					continue
				}

				content := cleanMessage(msg.Content, s)
				if content != "" && !strings.HasPrefix(content, "/") {
					ctx.GuildModel.Train(content)

					ctx.Mu.Lock()
					if ctx.ChannelModels[channelID] != nil {
						ctx.ChannelModels[channelID].Train(content)
					}
					ctx.Mu.Unlock()

					totalMessages++
				}
			}

			lastID = messages[len(messages)-1].ID
			time.Sleep(500 * time.Millisecond) // Rate limit
		}
	}

	return totalMessages
}

func RebuildModels(s *discordgo.Session, guildID string) {
	ctx := types.GetServerData(guildID)

	newGlobalModel := types.NewMarkovChain()
	newChannelModels := make(map[string]*types.MarkovChain)

	ctx.Mu.RLock()
	channels := make([]string, 0, len(ctx.WhitelistedChannels))
	for chanID := range ctx.WhitelistedChannels {
		channels = append(channels, chanID)
		newChannelModels[chanID] = types.NewMarkovChain()
	}
	ctx.Mu.RUnlock()

	for _, channelID := range channels {
		var lastID string

		for {
			messages, err := s.ChannelMessages(channelID, 100, lastID, "", "")
			if err != nil || len(messages) == 0 {
				break
			}

			for _, msg := range messages {
				ctx.Mu.RLock()
				optInTime, optedIn := ctx.OptedInUsers[msg.Author.ID]
				ctx.Mu.RUnlock()

				if !optedIn {
					continue
				}

				msgTime := msg.Timestamp

				if !optInTime.IsZero() && msgTime.Before(optInTime) {
					continue
				}

				content := cleanMessage(msg.Content, s)
				if content != "" && !strings.HasPrefix(content, "/") {
					newGlobalModel.Train(content)
					if newChannelModels[channelID] != nil {
						newChannelModels[channelID].Train(content)
					}
				}
			}

			lastID = messages[len(messages)-1].ID
			time.Sleep(500 * time.Millisecond) // Rate limit
		}
	}

	ctx.Mu.Lock()
	ctx.GuildModel = newGlobalModel
	ctx.ChannelModels = newChannelModels
	ctx.Mu.Unlock()
}
