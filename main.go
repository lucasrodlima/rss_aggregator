package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/lucasrodlima/rss_aggregator/internal/config"
	"github.com/lucasrodlima/rss_aggregator/internal/database"
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

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Items       []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
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

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(s *state, cmd command) error {
	return func(s *state, cmd command) error {
		currentUser, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}

		err = handler(s, cmd, currentUser)
		if err != nil {
			return err
		}
		return nil
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 3 {
		return fmt.Errorf("Enter name and url of feed as arguments")
	}

	name := cmd.args[1]
	url := cmd.args[2]

	newFeed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		return nil
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    newFeed.ID,
	})
	if err != nil {
		return nil
	}

	fmt.Printf("ID: %v\nCreatedAt: %v\nUpdatedAt: %v\nName: %v\nUrl: %v\nUserID: %v",
		newFeed.ID, newFeed.CreatedAt, newFeed.UpdatedAt, newFeed.Name, newFeed.Url, newFeed.UserID)

	return nil
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

func handlerAgg(s *state, cmd command) error {
	timeInput := cmd.args[1]

	time_between_reqs, err := time.ParseDuration(timeInput)
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeeds every %s\n", timeInput)

	for ; ; <-time.Tick(time_between_reqs) {
		err := scrapeFeeds(s)
		if err != nil {
			return err
		}
	}

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

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", "gator")

	newClient := http.DefaultClient
	res, err := newClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	xmlData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	feed := RSSFeed{}

	err = xml.Unmarshal(xmlData, &feed)
	if err != nil {
		return nil, err
	}

	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	for _, item := range feed.Channel.Items {
		item.Title = html.UnescapeString(item.Title)
		item.Description = html.UnescapeString(item.Description)
	}

	return &feed, nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		fmt.Printf("%v:\nURL: %v\nUser: %v\n\n", feed.Name, feed.Url, feed.User.String)
	}
	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	url := cmd.args[1]
	ctx := context.Background()

	feed, err := s.db.GetFeed(ctx, url)
	if err != nil {
		return err
	}

	newFeedFollow, err := s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return err
	}

	fmt.Printf("New follow added successfully:\nID: %v\nCreatedAt: %v\nUpdatedAt: %v\nUserID: %v\nFeedID: %v\n",
		newFeedFollow.ID, newFeedFollow.CreatedAt, newFeedFollow.UpdatedAt, newFeedFollow.UserID, newFeedFollow.FeedID)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	ctx := context.Background()
	username := s.cfg.CurrentUserName

	follows, err := s.db.GetFollowsForUser(ctx, user.ID)
	if err != nil {
		return err
	}

	fmt.Printf("%s (current user) is following:\n", username)
	for _, follow := range follows {
		fmt.Printf(" * %s\n", follow.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	err := s.db.DeleteFollow(context.Background(), database.DeleteFollowParams{
		Name: user.Name,
		Url:  cmd.args[1],
	})
	if err != nil {
		return err
	}

	fmt.Println("Follow removed")
	return nil
}

func scrapeFeeds(s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	err = s.db.MarkFeedFetched(context.Background(), database.MarkFeedFetchedParams{
		LastFetchedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		UpdatedAt: time.Now(),
		ID:        nextFeed.ID,
	})
	if err != nil {
		return err
	}

	rssFeed, err := fetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return nil
	}

	for _, item := range rssFeed.Channel.Items {
		_, err := s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Title:     item.Title,
			Url:       item.Link,
			Description: sql.NullString{
				String: item.Description,
				Valid:  true,
			},
			PublishedAt: sql.NullString{
				String: item.PubDate,
				Valid:  true,
			},
		})
		if err != nil {
			if strings.Contains(err.Error(), "posts_url_key") {
				fmt.Printf("post created - %s\n", item.Title)
				continue
			}
			return err
		}
		fmt.Printf("post created - %s\n", item.Title)
	}

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
	currentCommands.register("agg", handlerAgg)
	currentCommands.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	currentCommands.register("feeds", handlerFeeds)
	currentCommands.register("follow", middlewareLoggedIn(handlerFollow))
	currentCommands.register("following", middlewareLoggedIn(handlerFollowing))
	currentCommands.register("unfollow", middlewareLoggedIn(handlerUnfollow))

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
