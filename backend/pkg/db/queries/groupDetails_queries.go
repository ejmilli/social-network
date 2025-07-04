package db

import (
	"database/sql"
	"social-network/backend/pkg/db/sqlite"
	"social-network/backend/pkg/models"
	"social-network/backend/pkg/utils"
	"strings"
	"time"
)

// InsertGroupPost creates a new post in a group
func InsertGroupPost(tx *sql.Tx, userID, groupID int, title, content string) (int, error) {
	var postID int
	err := tx.QueryRow(`
		INSERT INTO group_posts (group_id, user_id, title, content, created_at)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id
	`, groupID, userID, title, content, time.Now()).Scan(&postID)
	return postID, err
}

// GetGroupPosts returns posts for a specific group
func GetGroupPosts(groupID, limit, offset int) ([]models.GroupPost, error) {
	query := `
		SELECT gp.id, gp.group_id, gp.user_id, u.nickname, gp.title, gp.content, 
			   gp.created_at, COALESCE(gp.votes, 0) as votes,
			   GROUP_CONCAT(pi.image_path) as image_paths
		FROM group_posts gp
		JOIN users u ON gp.user_id = u.id
		LEFT JOIN post_images pi ON gp.id = pi.post_id
		WHERE gp.group_id = ?
		GROUP BY gp.id
		ORDER BY gp.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := sqlite.GetDB().Query(query, groupID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.GroupPost
	for rows.Next() {
		var post models.GroupPost
		var imagePathsStr sql.NullString

		err := rows.Scan(&post.ID, &post.GroupID, &post.UserID, &post.Nickname,
			&post.Title, &post.Content, &post.CreatedAt, &post.Votes, &imagePathsStr)
		if err != nil {
			return nil, err
		}

		if imagePathsStr.Valid && imagePathsStr.String != "" {
			// Split comma-separated image paths
			post.ImagePaths = strings.Split(imagePathsStr.String, ",")
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// GetPostGroupID returns the group ID for a post (0 if not a group post)
func GetPostGroupID(postID int) (int, error) {
	var groupID sql.NullInt64

	// First check if it's a group post
	err := sqlite.GetDB().QueryRow(`
		SELECT group_id FROM group_posts WHERE id = ?
	`, postID).Scan(&groupID)

	if err != nil {
		if err == sql.ErrNoRows {
			// Not a group post, return 0
			return 0, nil
		}
		return 0, err
	}

	if groupID.Valid {
		return int(groupID.Int64), nil
	}
	return 0, nil
}

// Group Events Queries

// CreateGroupEvent creates a new event in a group
func CreateGroupEvent(groupID, userID int, title, description string, eventDate time.Time) (int, error) {
	var eventID int
	err := sqlite.GetDB().QueryRow(`
		INSERT INTO group_events (group_id, creator_id, title, description, event_date, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id
	`, groupID, userID, title, description, eventDate, time.Now()).Scan(&eventID)
	return eventID, err
}

// GetGroupEvents returns all events for a group
func GetGroupEvents(groupID int) ([]models.GroupEvent, error) {
	query := `
		SELECT ge.id, ge.group_id, ge.creator_id, u.nickname, ge.title, 
			   ge.description, ge.event_date, ge.created_at
		FROM group_events ge
		JOIN users u ON ge.creator_id = u.id
		WHERE ge.group_id = ?
		ORDER BY ge.event_date ASC
	`

	rows, err := sqlite.GetDB().Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.GroupEvent
	for rows.Next() {
		var event models.GroupEvent
		err := rows.Scan(&event.ID, &event.GroupID, &event.CreatorID,
			&event.CreatorName, &event.Title, &event.Description,
			&event.EventDate, &event.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Get responses for this event
		responses, err := getEventResponses(event.ID)
		if err != nil {
			return nil, err
		}
		event.Responses = responses

		events = append(events, event)
	}

	return events, nil
}

// getEventResponses is a helper function to get responses for an event
func getEventResponses(eventID int) ([]models.GroupEventResponse, error) {
	query := `
		SELECT ger.user_id, u.nickname, ger.response, ger.created_at
		FROM group_event_responses ger
		JOIN users u ON ger.user_id = u.id
		WHERE ger.event_id = ?
		ORDER BY ger.created_at DESC
	`

	rows, err := sqlite.GetDB().Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []models.GroupEventResponse
	for rows.Next() {
		var response models.GroupEventResponse
		err := rows.Scan(&response.UserID, &response.Nickname,
			&response.Response, &response.CreatedAt)
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}

	return responses, nil
}

// GetEventGroupID returns the group ID for an event
func GetEventGroupID(eventID int) (int, error) {
	var groupID int
	err := sqlite.GetDB().QueryRow(`
		SELECT group_id FROM group_events WHERE id = ?
	`, eventID).Scan(&groupID)
	return groupID, err
}

// RespondToEvent records a user's response to an event
func RespondToEvent(eventID, userID int, response string) error {
	// Use INSERT OR REPLACE to handle updating existing responses
	_, err := sqlite.GetDB().Exec(`
		INSERT OR REPLACE INTO group_event_responses (event_id, user_id, response, created_at)
		VALUES (?, ?, ?, ?)
	`, eventID, userID, response, time.Now())
	return err
}

// GetEventDetails returns detailed information about an event
func GetEventDetails(eventID, userID int) (*models.GroupEvent, error) {
	var event models.GroupEvent

	query := `
		SELECT ge.id, ge.group_id, ge.creator_id, u.nickname, ge.title, 
			   ge.description, ge.event_date, ge.created_at
		FROM group_events ge
		JOIN users u ON ge.creator_id = u.id
		WHERE ge.id = ?
	`

	err := sqlite.GetDB().QueryRow(query, eventID).Scan(
		&event.ID, &event.GroupID, &event.CreatorID, &event.CreatorName,
		&event.Title, &event.Description, &event.EventDate, &event.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Get all responses
	responses, err := getEventResponses(eventID)
	if err != nil {
		return nil, err
	}
	event.Responses = responses

	// Get user's response if exists
	var userResponse models.GroupEventResponse
	err = sqlite.GetDB().QueryRow(`
		SELECT user_id, response, created_at
		FROM group_event_responses
		WHERE event_id = ? AND user_id = ?
	`, eventID, userID).Scan(&userResponse.UserID, &userResponse.Response, &userResponse.CreatedAt)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == nil {
		// User has responded
		userResponse.Nickname = "" // We don't need this for user's own response
		event.UserResponse = &userResponse
	}

	return &event, nil
}

// Additional helper functions

// ValidateGroup validates group creation data
func ValidateGroup(title, description string) *utils.ValidationError {
	if title == "" || description == "" {
		return &utils.ValidationError{Message: "Title and description are required"}
	}
	if len(title) > 100 {
		return &utils.ValidationError{Message: "Title must be under 100 characters"}
	}
	if len(description) > 500 {
		return &utils.ValidationError{Message: "Description must be under 500 characters"}
	}
	return nil
}

// ValidateEvent validates event creation data
func ValidateEvent(title, description string, eventDate time.Time) *utils.ValidationError {
	if title == "" || description == "" {
		return &utils.ValidationError{Message: "Title and description are required"}
	}
	if len(title) > 100 {
		return &utils.ValidationError{Message: "Title must be under 100 characters"}
	}
	if len(description) > 500 {
		return &utils.ValidationError{Message: "Description must be under 500 characters"}
	}
	if eventDate.Before(time.Now()) {
		return &utils.ValidationError{Message: "Event date must be in the future"}
	}
	return nil
}
