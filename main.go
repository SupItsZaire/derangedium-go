package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/supitszaire/derangedium-go/commands"
	"github.com/supitszaire/derangedium-go/handlers"
	"github.com/supitszaire/derangedium-go/types"
)

func main() {
	// Load le funny .env file from current directory
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system/environment variables: BOT_TOKEN")
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("No token found, how do you expect this to work? ('export BOT_TOKEN=TOKEN' or 'BOT_TOKEN=TOKEN' in .env )") // vimae - token refrence????
	}
	dataFile := "data.json"
	if err := types.LoadData(dataFile); err != nil {
		log.Printf("Error loading data: %v", err)
	}

	saveTicker := time.NewTicker(5 * time.Minute)
	go func() {
		for range saveTicker.C {
			if err := types.SaveData(dataFile); err != nil {
				log.Printf("Error saving data: %v", err)
			} else {
				log.Println("Data saved success!")
			}
		}
	}()

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error starting/generating session:", err)
	}

	dg.AddHandler(handlers.HandleInteraction)
	dg.AddHandler(handlers.HandleMessage)

	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}
	defer dg.Close()

	log.Println("Registering commands...")
	commands.RegisterCommands(dg)

	log.Println("Derangedium-Go (V0.2.4a) is running. Press CTRL-C to exit.")
	log.Println("DEBUG: Starting log.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// thank you,
	// i'll say goodbye soon,
	// though it's the end of the world,
	// don't blame yourself

	// i really liked this refrence from the original Deuterium codebase
	// so i am stealing it >:) -claire

	log.Println("Shutting down, saving remaining data...")
	saveTicker.Stop()
	if err := types.SaveData(dataFile); err != nil {
		log.Printf("Error saving data on shutdown: %v", err)
	} else {
		log.Println("Remaining data saved successfully, shutting down...")
	}
}
