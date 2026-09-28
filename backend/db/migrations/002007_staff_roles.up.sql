CREATE TYPE staff_role AS ENUM ('admin', 'judge', 'moderator', 'lanista');
CREATE TABLE user_roles (
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 role staff_role NOT NULL,
 granted_by uuid REFERENCES users(id) ON DELETE SET NULL,
 granted_at timestamptz NOT NULL DEFAULT now(),
 reason text NOT NULL DEFAULT '',
 PRIMARY KEY (user_id, role)
);
INSERT INTO user_roles(user_id, role, reason)
 SELECT id, 'admin', 'Migrated staff access' FROM users WHERE is_admin;
INSERT INTO user_roles(user_id, role, reason)
 SELECT id, role, 'Migrated moderator access' FROM users
 CROSS JOIN unnest(ARRAY['judge','moderator']::staff_role[]) role
 WHERE is_moderator OR is_admin;
ALTER TABLE users DROP COLUMN is_admin, DROP COLUMN is_moderator;
