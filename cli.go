package main

import (
	"bekadoux/gator/internal/config"
	"bekadoux/gator/internal/database"
	"bekadoux/gator/internal/rss"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	handler, ok := c.handlers[cmd.name]
	if !ok {
		return fmt.Errorf("could not get %q from commands (perhaps it's unregistered?)", cmd.name)
	}
	err := handler(s, cmd)
	if err != nil {
		return err
	}

	return nil
}

func (c *commands) register(name string, f func(s *state, cmd command) error) {
	c.handlers[name] = f
}

func newCommands() *commands {
	return &commands{
		handlers: make(map[string]func(*state, command) error),
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: %v <name>", cmd.name)
	}

	userName := cmd.args[0]
	if exists, err := checkUserExists(s, userName); err == nil && !exists {
		return fmt.Errorf("user %q doesn't exist", userName)
	} else if err != nil {
		return err
	}

	if err := s.cfg.SetUser(cmd.args[0]); err != nil {
		return fmt.Errorf("%q: could not set user name: %w", cmd.name, err)
	}
	fmt.Println("The user name has been set.")

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: %v <name>", cmd.name)
	}

	userName := cmd.args[0]
	if exists, err := checkUserExists(s, userName); err == nil && exists {
		return fmt.Errorf("user %q already exists", userName)
	} else if err != nil {
		return err
	}

	userParams := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      userName,
	}

	_, err := s.db.CreateUser(context.Background(), userParams)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	fmt.Printf("User %q was created.\n", userName)

	if err := handlerLogin(s, cmd); err != nil {
		return fmt.Errorf("login after register: %w", err)
	}

	return nil
}

func handlerUsers(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: %v", cmd.name)
	}

	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("getting all users: %w", err)
	}
	if len(users) == 0 {
		fmt.Println("There are no users yet.")
		return nil
	}

	for _, user := range users {
		output := fmt.Sprintf("* %s", user.Name)
		if user.Name == s.cfg.CurrentUserName {
			output += " (current)"
		}
		fmt.Printf("%s\n", output)
	}

	return nil
}

func handlerReset(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: %v", cmd.name)
	}

	err := s.db.DeleteUsers(context.Background())
	if err != nil {
		return fmt.Errorf("deleting all users: %w", err)
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: %v <time between requests>", cmd.name)
	}

	timeBetweenReqs, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("could not parse time between requests for %s: %w", cmd.name, err)
	}

	fmt.Printf("Collecting feeds every %s...\n", timeBetweenReqs.String())
	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		err := scrapeFeeds(s)
		if err != nil {
			fmt.Printf("Could not fetch feed: %s", err.Error())
		}
	}

	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("usage: %v <feed name> <feed url>", cmd.name)
	}

	feedName, feedUrl := cmd.args[0], cmd.args[1]
	feedParams := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      feedName,
		Url:       feedUrl,
		UserID:    user.ID,
	}
	feed, err := s.db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		return fmt.Errorf("could not create feed: %w", err)
	}

	fmt.Printf("Feed %q at %s created successfully.\n", feed.Name, feed.Url)

	cmd.args = []string{feedUrl}
	err = handlerFollow(s, cmd, user)
	if err != nil {
		return fmt.Errorf("could not follow newly added feed %q: %w", feedName, err)
	}

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: %v", cmd.name)
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("getting all feeds: %w", err)
	}
	if len(feeds) == 0 {
		fmt.Println("No feeds added so far.")
		return nil
	}

	for _, feed := range feeds {
		creator, err := s.db.GetUserByID(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("could not get feed creator: %w", err)
		}
		creatorName := creator.Name

		output := fmt.Sprintf("* Name: %s\n", feed.Name)
		output += fmt.Sprintf("* URL: %s\n", feed.Url)
		output += fmt.Sprintf("* Creator Name: %s\n", creatorName)
		fmt.Printf("%s\n", output)
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: %v <feed url>", cmd.name)
	}

	feed, err := s.db.GetFeedByURL(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("could not get feed from database (search by URL): %w", err)
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("could not create feed follow database entry: %w", err)
	}

	fmt.Printf("%s is now following: %s\n", user.Name, feed.Name)

	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: %v", cmd.name)
	}

	feedFollows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf(
			"could not get feed follows for user %q from database: %w",
			s.cfg.CurrentUserName,
			err,
		)
	}

	fmt.Println("You're following:")
	for i, feedFollow := range feedFollows {
		fmt.Printf("  - %s", feedFollow.FeedName)
		if i < len(feedFollows)-1 {
			fmt.Printf("\n")
		}
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: %v <feed url>", cmd.name)
	}

	feedURL := cmd.args[0]
	feed, err := s.db.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("could not get feed from database (search by URL): %w", err)
	}

	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf(
			"could not delete feed follow from database (attempt to unfollow feed): %w",
			err,
		)
	}

	return nil
}

func middlewareLoggedIn(
	handler func(s *state, cmd command, user database.User) error,
) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("could not get current user from database: %w", err)
		}
		return handler(s, cmd, user)
	}
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("could not get next feed to fetch: %w", err)
	}

	fullFeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		return fmt.Errorf("could not fetch feed %q: %w", feed.Name, err)
	}
	fmt.Printf("%s Item Titles:\n", feed.Name)
	for _, item := range fullFeed.Channel.Item {
		fmt.Printf("  - %s\n", item.Title)
	}

	err = s.db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		return fmt.Errorf("could not mark feed %q as fetched: %w", feed.Name, err)
	}

	return nil
}

func checkUserExists(s *state, name string) (bool, error) {
	_, err := s.db.GetUser(context.Background(), name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("check whether user %q exists: %w", name, err)
	}
	return true, nil
}
