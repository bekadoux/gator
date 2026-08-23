-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: MarkFeedFetched :exec
UPDATE feeds
SET
    last_fetched_at = CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
    updated_at = CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT * FROM feeds
ORDER BY last_fetched_at ASC NULLS FIRST
LIMIT 1;

-- name: GetFeeds :many
SELECT * FROM feeds;

-- name: GetFeed :one
SELECT * FROM feeds
WHERE name = $1;

-- name: GetFeedByURL :one
SELECT * FROM feeds
WHERE url = $1;

-- name: DeleteFeeds :exec
DELETE FROM feeds;
