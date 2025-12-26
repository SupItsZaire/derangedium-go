package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "opt-in",
		Description: "Opt in to contribute your messages to the markov chain",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "backdate",
				Description: "Include all your past messages (default: false, only future messages)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "since",
				Description: "If backdating, start from this date (YYYY-MM-DD, e.g., 2022-02-02)",
				Required:    false,
			},
		},
	},
	{
		Name:        "opt-out",
		Description: "Opt out from contributing messages",
	},
	{
		Name:        "whitelist-channel",
		Description: "Add this channel to the training whitelist (Admin only)",
	},
	{
		Name:        "unwhitelist-channel",
		Description: "Remove this channel from the whitelist (Admin only)",
	},
	{
		Name:        "pretrain",
		Description: "Pre-train on past messages in this channel",
	},
	{
		Name:        "generate",
		Description: "Generate a message from the markov chain",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "scope",
				Description: "Use global or channel model",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Global", Value: "global"},
					{Name: "Channel", Value: "channel"},
				},
			},
		},
	},
	{
		Name:        "status",
		Description: "Show bot statistics",
	},
}

func RegisterCommands(s *discordgo.Session) {
	for _, cmd := range Commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Cannot create '%s' command: %v", cmd.Name, err)
		}
	}
}
