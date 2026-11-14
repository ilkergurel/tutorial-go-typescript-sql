-- LEARN(sql.security-ops.pooling-config): connections and memory are the two knobs to learn first.
-- context: when a change takes effect, and who may make it. boot_val is the
-- built-in default; the live values depend on your server, so they aren't shown.
SELECT name, context, boot_val, unit
FROM pg_settings
WHERE name IN ('max_connections', 'superuser_reserved_connections',
               'reserved_connections', 'shared_buffers', 'effective_cache_size',
               'work_mem', 'hash_mem_multiplier', 'maintenance_work_mem',
               'temp_file_limit', 'idle_in_transaction_session_timeout')
ORDER BY context, name;

-- context 'user': any session may change it for itself, even for one transaction.
SET LOCAL work_mem = '64MB';
SHOW work_mem;
SET LOCAL idle_in_transaction_session_timeout = '30s';
SHOW idle_in_transaction_session_timeout;

-- context 'postmaster': the memory and the process slots are set up at start.
SET LOCAL shared_buffers = '1GB';
