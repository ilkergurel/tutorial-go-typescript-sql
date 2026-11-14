-- LEARN(design.architecture.reliability): the server's half of an idempotency key, with two attempts racing.
-- A is the first attempt of a payment, B its retry; both carry the key k-7f3a.

A: BEGIN;
A: INSERT INTO idempotency_keys (key) VALUES ('k-7f3a') ON CONFLICT DO NOTHING RETURNING key;
B: BEGIN;
B: INSERT INTO idempotency_keys (key) VALUES ('k-7f3a') ON CONFLICT DO NOTHING RETURNING key;
A: WITH p AS (INSERT INTO payments (kurus) VALUES (125000) RETURNING id)
   UPDATE idempotency_keys SET payment_id = (SELECT id FROM p)
   WHERE key = 'k-7f3a' RETURNING payment_id;
A: COMMIT;
-- B got no row: someone else owns the key. Replay the stored result, charge nothing.
B: SELECT payment_id FROM idempotency_keys WHERE key = 'k-7f3a';
B: COMMIT;

-- Key k-9c1d: this time the first attempt fails before it commits.
A: BEGIN;
A: INSERT INTO idempotency_keys (key) VALUES ('k-9c1d') ON CONFLICT DO NOTHING RETURNING key;
B: BEGIN;
B: INSERT INTO idempotency_keys (key) VALUES ('k-9c1d') ON CONFLICT DO NOTHING RETURNING key;
A: ROLLBACK;
-- The key went with A's transaction, so B now owns it and does the work.
B: WITH p AS (INSERT INTO payments (kurus) VALUES (125000) RETURNING id)
   UPDATE idempotency_keys SET payment_id = (SELECT id FROM p)
   WHERE key = 'k-9c1d' RETURNING payment_id;
B: COMMIT;

A: SELECT k.key, k.payment_id, p.kurus
   FROM idempotency_keys AS k JOIN payments AS p ON p.id = k.payment_id ORDER BY k.key;
