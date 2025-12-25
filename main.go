package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/supitszaire/derangedium-go/commands"
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

	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	err = dg.Open()
	if err != nil {
		log.Fatal("Error establishing connection:", err)
	}
	defer dg.Close()

	log.Println("Registering commands, one minute.")
	commands.RegisterCommands(dg)
	log.Println("running, shutting up with SIGTERM")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
