package handlers

import (
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/supitszaire/derangedium-go/types"
)

func HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	ctx := types.GetServerData(m.GuildID)

	botMentioned := false
	for _, mention := range m.Mentions {
		if mention.ID == s.State.User.ID {
			botMentioned = true
			break
		}
	}

	isReplyToBot := false
	if m.ReferencedMessage != nil && m.ReferencedMessage.Author.ID == s.State.User.ID {
		isReplyToBot = true
	}

	if botMentioned || isReplyToBot {
		ctx.Mu.RLock()
		optedIn := ctx.OptedInUsers[m.Author.ID]
		ctx.Mu.RUnlock()

		if !optedIn {
			if rand.Intn(5) == 0 {
				s.ChannelMessageSend(m.ChannelID, "💡 Want to contribute to the markov chain? Opt in with `/opt-in`!")
			}
		}

		var model *types.MarkovChain
		ctx.Mu.RLock()
		if ctx.ChannelModels[m.ChannelID] != nil && rand.Intn(2) == 0 {
			model = ctx.ChannelModels[m.ChannelID]
		} else {
			model = ctx.GuildModel
		}
		ctx.Mu.RUnlock()

		if model != nil {
			text := model.Generate(50)
			if text != "" {
				s.ChannelMessageSend(m.ChannelID, text)
			}
		}
		return
	}

	ctx.Mu.RLock()
	whitelisted := ctx.WhitelistedChannels[m.ChannelID]
	optedIn := ctx.OptedInUsers[m.Author.ID]
	ctx.Mu.RUnlock()

	if !whitelisted || !optedIn {
		return
	}

	content := strings.TrimSpace(m.Content)
	if content == "" || strings.HasPrefix(content, "/") {
		return
	}

	ctx.GuildModel.Train(content)

	ctx.Mu.Lock()
	if ctx.ChannelModels[m.ChannelID] != nil {
		ctx.ChannelModels[m.ChannelID].Train(content)
	}
	ctx.Mu.Unlock()
}
