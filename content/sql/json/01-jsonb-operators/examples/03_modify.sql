CREATE TABLE project_settings (
  project text  PRIMARY KEY,
  prefs   jsonb NOT NULL
);
INSERT INTO project_settings VALUES
  ('TraceLite', '{"theme": "dark", "lang": "tr",
                  "notify": {"email": true, "slack": false}}');

-- || merges the top level only: the nested notify object is replaced whole.
SELECT prefs || '{"notify": {"slack": true}}' AS shallow_merge
FROM project_settings;

-- Change one nested value: a subscript assignment (PG 14) or jsonb_set.
UPDATE project_settings SET prefs['notify']['slack'] = 'true'
RETURNING prefs;
UPDATE project_settings SET prefs = jsonb_set(prefs, '{notify,sms}', 'false')
RETURNING prefs;

-- Delete a key with -, a nested one with #-.
UPDATE project_settings SET prefs = prefs - 'lang' #- '{notify,email}'
RETURNING prefs;

-- jsonb_set returns NULL when the new value is SQL NULL: an UPDATE built
-- this way nulls the whole document. jsonb_set_lax lets you choose.
SELECT jsonb_set(prefs, '{theme}', to_jsonb(NULL::text))          AS set_null,
       jsonb_set_lax(prefs, '{theme}', NULL)                      AS lax_default,
       jsonb_set_lax(prefs, '{theme}', NULL, true, 'delete_key')  AS lax_delete
FROM project_settings;
