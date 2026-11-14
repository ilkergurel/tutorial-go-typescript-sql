-- The limit is in bytes, not characters: ş takes two bytes in UTF-8.
SELECT length(repeat('ş', 4000)) AS characters, octet_length(repeat('ş', 4000)) AS bytes;

-- 7998 bytes fit, 8000 don't. The transaction is rolled back, so nobody is notified.
SELECT pg_notify('payload_limit_demo', repeat('ş', 3999));
SELECT pg_notify('payload_limit_demo', repeat('ş', 4000));
