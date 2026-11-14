CREATE TABLE requirements (
  code  text PRIMARY KEY,
  title text NOT NULL
);
INSERT INTO requirements VALUES ('REQ-1', 'Giriş ekranı'), ('REQ-2', 'Rapor; PDF');

-- TO STDOUT: the rows go to the client as COPY data, not as a result set.
-- psql prints them; this runner reads results the way a plain query call
-- does, so all it shows is the command tag with the row count.
COPY requirements TO STDOUT WITH (FORMAT csv, HEADER);

-- Any query can be the source: that is how you export a join or a view.
COPY (SELECT code FROM requirements WHERE title LIKE 'Rapor%') TO STDOUT;

-- A file name instead of STDIN/STDOUT: the server process opens the file,
-- on the server's disk, as the server's OS account. That takes a role.
SELECT current_user,
       pg_has_role('pg_read_server_files', 'USAGE')      AS read_files,
       pg_has_role('pg_write_server_files', 'USAGE')     AS write_files,
       pg_has_role('pg_execute_server_program', 'USAGE') AS run_programs;

COPY requirements FROM 'C:/Users/ayse/Downloads/requirements.csv' WITH (FORMAT csv, HEADER);
