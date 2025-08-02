# Gator: A CLI-Based RSS Aggregator

Gator is a command-line RSS aggregator written in Go. It allows you to subscribe to and manage multiple RSS feeds, browse articles, and read them directly from your terminal.

## Features

- **User Management**: Create and switch between multiple user accounts.
- **Feed Management**: Add, list, and manage your RSS feed subscriptions.
- **Article Browsing**: Browse and read articles from your subscribed feeds.
- **Background Aggregation**: Automatically fetch new articles from your feeds at regular intervals.

## Project Structure

The project is organized into the following directories:

- **`main.go`**: Contains the main application logic, including command handlers and middleware.
- **`internal`**: Houses the core business logic, database interactions, and configuration management.
- **`sql`**: Defines the database schema and SQL queries for interacting with the database.

## Getting Started

### Prerequisites

- Go 1.16 or higher
- PostgreSQL

### Installation

1. **Clone the repository:**

   ```bash
   git clone https://github.com/lucasrodlima/gator.git
   cd gator
   ```

2. **Install dependencies:**

   ```bash
   go mod tidy
   ```

3. **Set up the database:**

   - Create a PostgreSQL database for Gator.
   - Create a `.gatorconfig.json` file in your home directory with the following content:

     ```json
     {
       "db_url": "postgres://<user>:<password>@<host>:<port>/<database>",
       "current_username": ""
     }
     ```

4. **Build the application:**

   ```bash
   go build -o gator .
   ```

## Usage

Gator provides a set of commands to interact with the application. Here are some of the most common commands:

- **`register <username>`**: Create a new user account.
- **`login <username>`**: Log in to an existing user account.
- **`addfeed <name> <url>`**: Add a new RSS feed to your subscriptions.
- **`feeds`**: List all available RSS feeds.
- **`follow <url>`**: Follow a feed to start receiving articles from it.
- **`following`**: List all the feeds you are currently following.
- **`unfollow <url>`**: Unfollow a feed.
- **`browse [limit]`**: Browse articles from your followed feeds.
- **`agg <duration>`**: Start the background aggregator to fetch new articles every `<duration>`.

For more information on a specific command, use the `help` command:

```bash
./gator help <command>
```

