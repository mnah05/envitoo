ALTER TABLE events ALTER COLUMN slug DROP DEFAULT;

ALTER TABLE events ALTER COLUMN slug SET NOT NULL;

ALTER TABLE volunteers
ADD CONSTRAINT volunteers_role_check CHECK (
    role IN (
        'coordinator',
        'security',
        'host',
        'photographer',
        'logistics'
    )
);