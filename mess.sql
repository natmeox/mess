CREATE TABLE character (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE account (
    loginname TEXT NOT NULL PRIMARY KEY,
    passwordhash TEXT NOT NULL,
    character INTEGER NOT NULL REFERENCES character,
    created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE TABLE room (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    creator INTEGER NOT NULL REFERENCES character,
    created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE TABLE exit (
    command TEXT NOT NULL,
    fromroom INTEGER NOT NULL REFERENCES room,
    toroom INTEGER NOT NULL REFERENCES room
);