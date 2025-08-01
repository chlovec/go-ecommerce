CREATE TABLE IF NOT EXISTS categories (
    id bigserial PRIMARY KEY,
    created_at timestamptz(0) NOT NULL DEFAULT NOW(),
    name text NOT NULL,
    description text NOT NULL,
    version integer NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS products (
    id bigserial PRIMARY KEY,
    created_at timestamptz(0) NOT NULL DEFAULT NOW(),
    name text NOT NULL,
    description text NOT NULL,
    category_id bigint NOT NULL REFERENCES categories,
    version integer NOT NULL DEFAULT 1
);
