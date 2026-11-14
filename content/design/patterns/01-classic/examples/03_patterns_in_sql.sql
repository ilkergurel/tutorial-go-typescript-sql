-- LEARN(design.patterns.classic): strategy as rows, adapter as a view, observer as a trigger.

-- Strategy as data: each fee rule is a row, and choosing one is a join.
CREATE TABLE fee_rules (
    name            text PRIMARY KEY,
    base_kurus      int NOT NULL,
    per_desi_kurus  int NOT NULL,
    free_over_kurus int              -- NULL: never free
);
INSERT INTO fee_rules VALUES ('flat', 4990, 0, NULL), ('carrier', 3990, 1250, 50000);

-- Adapter: a legacy table in someone else's shape...
CREATE TABLE siparis (no int PRIMARY KEY, desi numeric NOT NULL,
                      tutar_tl numeric(10, 2) NOT NULL, kargo text NOT NULL);
INSERT INTO siparis VALUES (1, 2, 450.00, 'flat'), (2, 6, 720.00, 'carrier'),
                           (3, 2, 450.00, 'carrier');
-- ...and a view in the shape every consumer expects.
CREATE VIEW orders AS
SELECT no AS id, desi, (tutar_tl * 100)::int AS total_kurus, kargo AS rule
FROM siparis;

SELECT o.id, o.rule,
       CASE WHEN o.total_kurus >= r.free_over_kurus THEN 0
            ELSE r.base_kurus + r.per_desi_kurus * o.desi END::int AS fee_kurus
FROM orders AS o JOIN fee_rules AS r ON r.name = o.rule
ORDER BY o.id;

-- Observer: a trigger hears every insert, whoever makes it, and records an
-- event in an outbox table in the same transaction.
CREATE TABLE outbox (
    id      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic   text NOT NULL,
    payload jsonb NOT NULL
);
CREATE FUNCTION publish_order() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO outbox (topic, payload)
    VALUES ('order.placed', jsonb_build_object('id', NEW.no, 'tl', NEW.tutar_tl));
    RETURN NULL; -- the result of an AFTER ... FOR EACH ROW trigger is ignored
END $$;
CREATE TRIGGER siparis_placed AFTER INSERT ON siparis
    FOR EACH ROW EXECUTE FUNCTION publish_order();

INSERT INTO siparis VALUES (4, 1, 99.90, 'flat'), (5, 3, 1250.00, 'carrier');
SELECT id, topic, payload FROM outbox ORDER BY id;
