package handlers

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/supitszaire/derangedium-go/types"
)

func calculateModelSize(model *types.MarkovChain) int {
	model.Mu.RLock()
	defer model.Mu.RUnlock()

	size := 0
	for key, values := range model.Chain {
		size += len(key) // key string
		for _, val := range values {
			size += len(val) // value strings
		}
		size += len(values) * 8 // slice overhead (approximate)
	}
	return size
}

func HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fmt.Printf("Interaction received: Type=%v\n", i.Type)

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	fmt.Printf("Command received: %s\n", data.Name)

	guildID := i.GuildID
	userID := i.Member.User.ID
	ctx := types.GetServerData(guildID)

	switch data.Name {
	case "opt-in":
		backdate := false
		var sinceDate time.Time

		// parsing :3
		for _, opt := range data.Options {
			switch opt.Name {
			case "backdate":
				backdate = opt.BoolValue()
			case "since":
				parsed, err := time.Parse("2001-09-11", opt.StringValue())
				if err != nil {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "❌ Invalid date format. Use YYYY-MM-DD (e.g., 2022-02-02)",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					return
				}
				sinceDate = parsed
			}
		}

		// setting opt in time
		optInTime := time.Now()
		if backdate {
			optInTime = time.Time{} // zero time = beginning of Discord history (if possible)
		}
		if !sinceDate.IsZero() {
			optInTime = sinceDate
		}

		ctx.Mu.Lock()
		ctx.OptedInUsers[userID] = optInTime
		ctx.Mu.Unlock()

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})

		go func() {
			trained := PretrainUser(s, guildID, userID, optInTime)

			msg := fmt.Sprintf("✅ You're now opted in! Pre-trained on %d of your past messages.", trained)
			if backdate {
				if !sinceDate.IsZero() {
					msg = fmt.Sprintf("✅ You're now opted in! Pre-trained on %d messages since %s.", trained, sinceDate.Format("2016-01-02"))
				} else {
					msg = fmt.Sprintf("✅ You're now opted in with full backdate! Pre-trained on %d of your past messages.", trained)
				}
			}

			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}()

	case "opt-out":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})

		ctx.Mu.Lock()
		delete(ctx.OptedInUsers, userID)
		ctx.Mu.Unlock()

		// rebuild to remove user messages
		go func() {
			RebuildModels(s, guildID)
			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: "✅ You've been opted out. All models have been rebuilt without your messages.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}()

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
				Content: "✅ This channel is now whitelisted for training!",
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
				Content: "✅ This channel is no longer whitelisted.",
			},
		})

	case "pretrain":
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
					Content: "⚠️ This channel isn't whitelisted. Use /whitelist-channel first to initiate pre-training.",
				})
				return
			}
			ctx.Mu.RUnlock()

			total := Pretrain(s, guildID, channelID)

			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: fmt.Sprintf("✅ Pre-training complete! Processed %d messages.", total),
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

		globalBytes := calculateModelSize(ctx.GuildModel)
		channelBytes := 0
		for _, model := range ctx.ChannelModels {
			channelBytes += calculateModelSize(model)
		}
		ctx.Mu.RUnlock()

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		content := fmt.Sprintf("**Derangedium-Go (V0.2.4a) by GitHub.com/SupitsZaire**\n"+
			"Opted-in users: %d\n"+
			"Whitelisted channels: %d\n"+
			"Global model: %d word pairs (%.2f MiB)\n"+
			"Channel models: %.2f MiB total\n"+
			"RAM usage: %.2f MiB allocated, %.2f MiB in use",
			optedIn, whitelisted, globalSize,
			float64(globalBytes)/1024/1024,
			float64(channelBytes)/1024/1024,
			float64(m.Alloc)/1024/1024,
			float64(m.Sys)/1024/1024)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}
