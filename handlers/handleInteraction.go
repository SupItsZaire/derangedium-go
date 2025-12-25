package handlers

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/supitszaire/derangedium-go/types"
)

func HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	guildID := i.GuildID
	userID := i.Member.User.ID
	ctx := types.GetServerData(guildID)

	switch data.Name {
	case "opt-in":
		ctx.Mu.Lock()
		ctx.OptedInUsers[userID] = true
		ctx.Mu.Unlock()

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ You're now opted in! Your messages will help train the markov chain.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

	case "opt-out":
		ctx.Mu.Lock()
		delete(ctx.OptedInUsers, userID)
		ctx.Mu.Unlock()

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ You've been opted out. Your future messages won't be used for training.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

	case "whitelist-channel":
		channelID := i.ChannelID
		ctx.Mu.Lock()
		ctx.WhitelistedChannels[channelID] = true
		if ctx.ChannelModels[channelID] == nil {
			ctx.ChannelModels[channelID] = types.NewMarkovChain()
		}
		ctx.Mu.Unlock()

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ This channel is now whitelisted for training! Opted-In users are now contributing to training data.",
			},
		})

	case "unwhitelist-channel":
		channelID := i.ChannelID
		ctx.Mu.Lock()
		delete(ctx.WhitelistedChannels, channelID)
		ctx.Mu.Unlock()

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This channel is no longer whitelisted. Training will no longer occur here!",
			},
		})

	case "pretrain":
		// Defer the response since pretrain takes time
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})

		go func() {
			channelID := i.ChannelID
			ctx := types.GetServerData(guildID)

			ctx.Mu.RLock()
			if !ctx.WhitelistedChannels[channelID] {
				ctx.Mu.RUnlock()
				s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
					Content: "⚠️ This channel isn't whitelisted. Use /whitelist-channel first to train!",
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				return
			}
			ctx.Mu.RUnlock()

			total := Pretrain(s, guildID, channelID)

			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: fmt.Sprintf("Pre-training complete! Processed a total of %d messages.", total),
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}()

	case "generate":
		scope := "global"
		if len(data.Options) > 0 {
			scope = data.Options[0].StringValue()
		}

		var model *types.MarkovChain
		ctx.Mu.RLock()
		if scope == "channel" {
			model = ctx.ChannelModels[i.ChannelID]
		} else {
			model = ctx.GuildModel
		}
		ctx.Mu.RUnlock()

		text := ""
		if model != nil {
			text = model.Generate(50)
		}

		if text == "" {
			text = "Not enough training data yet! Try /pretrain or chat more in whitelisted channels."
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: text,
			},
		})

	case "status":
		ctx.Mu.RLock()
		optedIn := len(ctx.OptedInUsers)
		whitelisted := len(ctx.WhitelistedChannels)
		globalSize := len(ctx.GuildModel.Chain)
		ctx.Mu.RUnlock()

		content := fmt.Sprintf("**derangedium-go v0.1.1a stats**\n"+
			"Opted-in users: %d\n"+
			"Whitelisted channels: %d\n"+
			"Global model size: %d word pairs",
			optedIn, whitelisted, globalSize)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}
