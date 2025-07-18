CREATE TABLE
    IF NOT EXISTS group_events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        group_id INTEGER NOT NULL,
        creator_id INTEGER NOT NULL,
        title TEXT NOT NULL,
        description TEXT NOT NULL,
        event_date DATETIME NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE,
        FOREIGN KEY (creator_id) REFERENCES users (id) ON DELETE CASCADE
    );