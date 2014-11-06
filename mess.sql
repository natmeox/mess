CREATE TYPE thingtype AS ENUM ('thing', 'place', 'player', 'action', 'program');

CREATE TABLE thing (
    id SERIAL PRIMARY KEY,
    type thingtype NOT NULL DEFAULT 'thing',
    name TEXT NOT NULL,
    creator INTEGER REFERENCES thing DEFERRABLE INITIALLY DEFERRED,
    created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    owner INTEGER REFERENCES thing DEFERRABLE INITIALLY DEFERRED,
    accesslist INTEGER[] NOT NULL DEFAULT ARRAY[]::integer[],
    parent INTEGER REFERENCES thing,
    tabledata JSON NOT NULL DEFAULT '{}'::json
);

CREATE TABLE account (
    loginname TEXT NOT NULL PRIMARY KEY,
    passwordhash TEXT NOT NULL,
    character INTEGER NOT NULL REFERENCES thing,
    created TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);
