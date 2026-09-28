-- Sender presentation is independent from the private-message audience.
ALTER TABLE chat_messages ADD COLUMN sender_team_id gd_team_id;
UPDATE chat_messages SET sender_team_id=team_id WHERE team_id IS NOT NULL;
