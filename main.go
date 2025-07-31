package main

import _ "github.com/lib/pq"

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/lucasrodlima/rss_aggregator/internal/config"
	"github.com/lucasrodlima/rss_aggregator/internal/database"
	"os"
	"time"
)

type state struct {
	cfg *config.Config
	db  *database.Queries
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	handlerFunc, ok := c.handlers[cmd.name]
	if !ok {
		fmt.Println("Non existent command")
		os.Exit(1)
	}

	err := handlerFunc(s, cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *commands) register(name string, f func(s *state, cmd command) error) {
	c.handlers[name] = f
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Println("Error retrieving all users")
		os.Exit(1)
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf(" * %s (current)\n", user.Name)
		} else {
			fmt.Printf(" * %s\n", user.Name)
		}
	}
	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		fmt.Println("Error resetting database")
		os.Exit(1)
	}

	fmt.Println("Database reset!")

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("Username is required")
	}

	username := cmd.args[1]

	_, err := s.db.GetUser(context.Background(), username)
	if err == nil {
		fmt.Printf("User %s already exists, use command \"login\"\n", username)
		os.Exit(1)
	}

	newUser, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	})
	if err != nil {
		return err
	}

	err = s.cfg.SetUser(newUser.Name)
	if err != nil {
		fmt.Printf("Error setting new user\n")
		os.Exit(1)
	}

	fmt.Printf("User %s created successfully\n", newUser.Name)

	fmt.Printf("ID: %v\nCreatedAt: %v\nUpdatedAt: %v\nName: %v\n",
		newUser.ID, newUser.CreatedAt, newUser.UpdatedAt, newUser.Name)

	return nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("Username is required")
	}

	username := cmd.args[1]

	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		fmt.Printf("User %s doesn't exist\n", username)
		os.Exit(1)
	}

	err = s.cfg.SetUser(username)
	if err != nil {
		return fmt.Errorf("Error setting new user")
	}

	fmt.Printf("Login successful as %s!\n", username)
	return nil
}

func main() {
	sysConfig, err := config.Read()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	currentDb, err := sql.Open("postgres", sysConfig.DbUrl)

	dbQueries := database.New(currentDb)

	currentState := state{
		cfg: sysConfig,
		db:  dbQueries,
	}

	currentCommands := commands{
		handlers: make(map[string]func(*state, command) error),
	}

	currentCommands.register("login", handlerLogin)
	currentCommands.register("register", handlerRegister)
	currentCommands.register("reset", handlerReset)
	currentCommands.register("users", handlerUsers)

	currentArgs := os.Args
	if len(currentArgs) < 2 {
		fmt.Println("Not enough arguments were provided")
		os.Exit(1)
	}

	currentCommand := command{
		name: currentArgs[1],
		args: currentArgs[1:],
	}

	err = currentCommands.run(&currentState, currentCommand)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
