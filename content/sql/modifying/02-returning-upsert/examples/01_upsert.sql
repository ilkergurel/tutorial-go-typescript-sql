-- LEARN(sql.modifying.returning-upsert): atomic upsert + read-back in one statement.

CREATE TABLE stock (sku text PRIMARY KEY, qty int NOT NULL CHECK (qty >= 0));
INSERT INTO stock VALUES ('widget', 10), ('gadget', 3);

-- Deliver a shipment: insert new skus, add to existing ones.
INSERT INTO stock (sku, qty)
VALUES ('widget', 5), ('gizmo', 7)
ON CONFLICT (sku) DO UPDATE
SET qty = stock.qty + EXCLUDED.qty
RETURNING sku, qty;

-- MERGE: same shape, but free to also delete or conditionally update.
CREATE TABLE incoming (sku text, qty int);
INSERT INTO incoming VALUES ('gadget', -2), ('doodad', 1);

MERGE INTO stock s
USING incoming i ON s.sku = i.sku
WHEN MATCHED AND s.qty + i.qty <= 0 THEN DELETE
WHEN MATCHED THEN UPDATE SET qty = s.qty + i.qty
WHEN NOT MATCHED THEN INSERT (sku, qty) VALUES (i.sku, i.qty)
RETURNING merge_action(), s.sku AS sku, s.qty AS qty;
