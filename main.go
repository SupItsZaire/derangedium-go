package main

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/supitszaire/derangedium-go/handlers"
)

func main() {

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("stupid bitch set yo token :)")
	}
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating session? i dunno, maybe this helps:", err)
	}

	dg.AddHandler(handlers.HandleMessage)
	dg.AddHandler(handlers.HandleMessage)
}
