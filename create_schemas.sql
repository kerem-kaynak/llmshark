-- Create additional schemas
CREATE SCHEMA IF NOT EXISTS blog;
CREATE SCHEMA IF NOT EXISTS shop;

-- Create tables in the test schema (adding to existing ones)
CREATE TABLE IF NOT EXISTS test.comments (
    id integer PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    post_id integer NOT NULL REFERENCES test.posts(id),
    user_id integer NOT NULL REFERENCES test.users(id),
    content text NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS test.categories (
    id integer PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name varchar(100) NOT NULL UNIQUE,
    description text
);

-- Create tables in blog schema
CREATE TABLE IF NOT EXISTS blog.articles (
    id integer PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    title varchar(200) NOT NULL,
    content text,
    author_id integer NOT NULL,
    published boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS blog.tags (
    id integer PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name varchar(50) NOT NULL UNIQUE
);

-- Create tables in shop schema
CREATE TABLE IF NOT EXISTS shop.products (
    id integer PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name varchar(200) NOT NULL,
    description text,
    price decimal(10,2) NOT NULL,
    stock integer NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS shop.orders (
    id integer PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id integer NOT NULL,
    total_amount decimal(10,2) NOT NULL,
    status varchar(50) NOT NULL DEFAULT 'pending',
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);

-- Add some comments for testing
COMMENT ON TABLE test.comments IS 'User comments on posts';
COMMENT ON TABLE blog.articles IS 'Blog articles and content';
COMMENT ON TABLE shop.products IS 'Product catalog';
COMMENT ON COLUMN shop.products.price IS 'Product price in USD';
