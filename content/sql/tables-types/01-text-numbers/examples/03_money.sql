-- A literal with a decimal point is numeric (exact) unless you cast it.
SELECT pg_typeof(0.1) AS literal_type,
       0.1 + 0.2 = 0.3 AS numeric_equal,
       0.1::float8 + 0.2::float8 = 0.3::float8 AS float_equal,
       0.1::float8 + 0.2::float8 AS float_sum;

CREATE TABLE payments (
  amount numeric(12, 2)   NOT NULL,  -- exact: 10 digits before the point, 2 after
  approx double precision NOT NULL   -- the mistake, kept for comparison
);

-- Ten payments of 0.10 TL.
INSERT INTO payments SELECT 0.10, 0.10 FROM generate_series(1, 10);

SELECT sum(amount) AS exact_total,
       sum(approx) AS float_total,
       sum(approx) = 1 AS float_total_is_one
FROM payments;

-- numeric(12, 2) rounds extra decimals on the way in, ties away from zero.
-- round() on a float rounds ties to even (on most platforms).
SELECT 19.995::numeric(12, 2) AS stored_as,
       round(2.5) AS numeric_round,
       round(2.5::float8) AS float_round;

-- Too many digits before the point is an error, never a silent wrap.
INSERT INTO payments VALUES (12345678901.00, 0);
