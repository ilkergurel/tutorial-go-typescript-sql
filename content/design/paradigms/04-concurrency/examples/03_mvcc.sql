-- LEARN(design.paradigms.concurrency): transactions run at once on snapshots; only writers of the same row wait for each other.
-- A: and B: are two connections, two buyers. Each product has one item left.
A: CREATE TABLE stock (sku text PRIMARY KEY, left_qty int NOT NULL CHECK (left_qty >= 0));
A: INSERT INTO stock VALUES ('racy', 1), ('fixed', 1);

-- Racy: read, decide in the application, write the value it computed.
A: BEGIN;
B: BEGIN;
A: SELECT left_qty FROM stock WHERE sku = 'racy';
B: SELECT left_qty FROM stock WHERE sku = 'racy';
A: UPDATE stock SET left_qty = 0 WHERE sku = 'racy';
B: UPDATE stock SET left_qty = 0 WHERE sku = 'racy';
A: COMMIT;
B: COMMIT;

-- Fixed: the check is part of the write.
A: BEGIN;
B: BEGIN;
A: UPDATE stock SET left_qty = left_qty - 1 WHERE sku = 'fixed' AND left_qty > 0;
B: SELECT left_qty FROM stock WHERE sku = 'fixed';
B: UPDATE stock SET left_qty = left_qty - 1 WHERE sku = 'fixed' AND left_qty > 0;
A: COMMIT;
B: COMMIT;
