CREATE TABLE IF NOT EXISTS group_invitations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id    INTEGER NOT NULL,
    inviter_id  INTEGER NOT NULL,
    invitee_id  INTEGER NOT NULL,
    status      TEXT NOT NULL CHECK (status IN ('pending','accepted','declined')),
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id)   REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_id) REFERENCES users(id)  ON DELETE CASCADE,
    FOREIGN KEY (invitee_id) REFERENCES users(id)  ON DELETE CASCADE
);