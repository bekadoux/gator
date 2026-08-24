# Gator

Gator is a command-line RSS feed aggregator backed by PostgreSQL. It supports
multiple application users, shared feeds, feed following, periodic collection,
and browsing saved posts.

This project was built as part of the
[Boot.dev RSS Aggregator course](https://www.boot.dev/courses/build-blog-aggregator-golang).

## Setup

### Install Prerequisites

Make sure you have the following installed before setting up Gator:

- [Go](https://go.dev/doc/install) 1.26.4 or later
- [PostgreSQL](https://www.postgresql.org/download/), with the server running
- [Goose](https://github.com/pressly/goose) for database migrations
- Git, to download the migration files

sqlc is used to generate the database access code, but it is not required to
install or run Gator.

You can use your operating system's package manager for these dependencies. For
example, on macOS with Homebrew:

```sh
brew install go git postgresql@17 goose
brew services start postgresql@17
```

On Ubuntu or Debian:

```sh
sudo apt update
sudo apt install git golang-go postgresql postgresql-client
sudo systemctl enable --now postgresql
```

Check `go version` after installation and use
the official Go installer if your package manager provides an older release.

### Configure PATH

Go installs command-line programs in `$(go env GOPATH)/bin` by default. Add
that directory to your `PATH` for the current shell if it is not already
available:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

To make this change permanent for Bash, add it to `~/.bashrc` and reload the
file:

```sh
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> "$HOME/.bashrc"
source "$HOME/.bashrc"
```

Use your shell's equivalent configuration file, such as `~/.zshrc` for Zsh.

### Install Gator and Goose

Install the Gator CLI with Go:

```sh
go install github.com/bekadoux/gator@latest
```

If Goose was not installed through a package manager, install it with Go:

```sh
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### Clone the Repository

Clone the repository so Goose can access the migration files:

```sh
git clone https://github.com/bekadoux/gator.git
cd gator
```

### Set Up PostgreSQL

PostgreSQL roles are database accounts. They are separate from the application
users created later with `gator register`.

Create a dedicated PostgreSQL role and database. These commands connect through
the default PostgreSQL administrator role, `postgres`:

```sh
createuser --username postgres --pwprompt gator
createdb --username postgres --owner=gator gator
```

Depending on how PostgreSQL was installed, you may need to run these commands as
the `postgres` operating-system user, use a different administrator role, or
provide additional connection options.

Set a connection URL using the password entered for the `gator` role. Replace
`YOUR_PASSWORD` before running the command. URL-encode special characters in the
password.

```sh
export GATOR_DB_URL='postgres://gator:YOUR_PASSWORD@localhost:5432/gator?sslmode=disable'
```

The `sslmode=disable` setting is intended for a local development database. Use
the SSL settings required by your PostgreSQL server when connecting remotely.

`GATOR_DB_URL` only needs to exist while running Goose and creating the config
file below. It contains a database password, so keeping it session-only is safer
than adding it permanently to `~/.bashrc`.

Apply all migrations from the repository root:

```sh
goose -dir ./sql/schema postgres "$GATOR_DB_URL" up
```

You can confirm their status with:

```sh
goose -dir ./sql/schema postgres "$GATOR_DB_URL" status
```

### Configure Gator

Gator reads its configuration from `~/.gatorconfig.json`. Create it using the
same database URL used by Goose:

```sh
echo "{\"db_url\":\"${GATOR_DB_URL}\",\"current_user_name\":\"\"}" > "$HOME/.gatorconfig.json"
chmod 600 "$HOME/.gatorconfig.json"
```

The username starts empty. The `register` and `login` commands update
`current_user_name` automatically.

After creating the config file, you can remove the temporary database variable
from the current shell:

```sh
unset GATOR_DB_URL
```

### Quick Start

Create and select your first Gator user:

```sh
gator register your-username
```

Add an RSS feed. The user who adds a feed follows it automatically:

```sh
gator addfeed "Lane's Blog" "https://www.wagslane.dev/index.xml"
```

Start the aggregator with a positive Go duration such as `30s`, `1m`, or `1h`:

```sh
gator agg 1m
```

The aggregator fetches one feed immediately, then one feed per interval. Leave
it running to collect posts and stop it with `Ctrl+C`. In another terminal, or
after stopping the aggregator, browse the saved posts from feeds you follow. A
failed feed is retried on a later rotation without blocking the other feeds.

```sh
gator browse 5
```

## Available Commands

| Command                      | Description                                                                                |
| ---------------------------- | ------------------------------------------------------------------------------------------ |
| `gator register <name>`      | Create an application user and select it as the current user.                              |
| `gator login <name>`         | Select an existing application user.                                                       |
| `gator users`                | List all application users and identify the current user.                                  |
| `gator addfeed <name> <url>` | Add a feed and follow it as the current user. Quote names containing spaces.               |
| `gator feeds`                | List all feeds and their owners.                                                           |
| `gator follow <url>`         | Follow an existing feed by its exact URL.                                                  |
| `gator unfollow <url>`       | Stop following a feed.                                                                     |
| `gator following`            | List feeds followed by the current user.                                                   |
| `gator agg <duration>`       | Collect feeds continuously at the requested interval.                                      |
| `gator browse [limit]`       | Show recent posts from followed feeds. The default limit is 2.                             |
| `gator reset`                | Delete all application users. This is destructive and intended for local development only. |
