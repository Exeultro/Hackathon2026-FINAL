-- NULL = shared (visible to everyone); otherwise "user:<id>" or an anonymous X-Client-Id
ALTER TABLE lectures ADD COLUMN owner TEXT;
ALTER TABLE courses ADD COLUMN owner TEXT;
CREATE INDEX lectures_owner_idx ON lectures(owner);
CREATE INDEX courses_owner_idx ON courses(owner);
