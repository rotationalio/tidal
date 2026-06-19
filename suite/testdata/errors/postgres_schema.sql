CREATE TABLE IF NOT EXISTS authors (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS books (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author_id INTEGER NOT NULL REFERENCES authors(id) ON DELETE RESTRICT,
    price DECIMAL(10, 2) DEFAULT NULL CHECK (price > 0)
);

INSERT INTO authors (name, email) VALUES ('F. Scott Fitzgerald', 'f.scott.fitzgerald@example.com');
INSERT INTO authors (name, email) VALUES ('J.D. Salinger', 'j.d.salinger@example.com');

INSERT INTO books (title, author_id) VALUES ('The Great Gatsby', (SELECT id FROM authors WHERE name = 'F. Scott Fitzgerald'));
INSERT INTO books (title, author_id) VALUES ('The Beautiful and Damned', (SELECT id FROM authors WHERE name = 'F. Scott Fitzgerald'));
INSERT INTO books (title, author_id) VALUES ('The Catcher in the Rye', (SELECT id FROM authors WHERE name = 'J.D. Salinger'));
INSERT INTO books (title, author_id) VALUES ('Nine Stories', (SELECT id FROM authors WHERE name = 'J.D. Salinger'));
INSERT INTO books (title, author_id) VALUES ('Franny and Zooey', (SELECT id FROM authors WHERE name = 'J.D. Salinger'));