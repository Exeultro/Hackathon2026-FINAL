DROP INDEX IF EXISTS courses_owner_idx;
DROP INDEX IF EXISTS lectures_owner_idx;
ALTER TABLE courses DROP COLUMN IF EXISTS owner;
ALTER TABLE lectures DROP COLUMN IF EXISTS owner;
