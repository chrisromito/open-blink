-- init-databases.sql

-- Create the openblink database
CREATE DATABASE openblink;

-- Create the test_openblink database
CREATE DATABASE test_openblink;

-- Connect to openblink database and create schema
\c openblink;
\i /docker-entrypoint-initdb.d/01-schema.sql

-- Connect to test_openblink database and create schema
\c test_openblink;
\i /docker-entrypoint-initdb.d/01-schema.sql