package handlers

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/supitszaire/derangedium-go/types"
)

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
			if msg.Author.Bot {
				continue
			}

			ctx.Mu.RLock()
			optedIn := ctx.OptedInUsers[msg.Author.ID]
			ctx.Mu.RUnlock()

			if !optedIn {
				continue
			}

			content := strings.TrimSpace(msg.Content)
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
