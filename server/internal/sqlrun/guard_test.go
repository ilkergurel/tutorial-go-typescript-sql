package sqlrun

import "testing"

func TestRefuseTxControl(t *testing.T) {
	for stmt, refuse := range map[string]bool{
		"COMMIT":                         true,
		"commit and chain":               true,
		"-- done\nCOMMIT":                true,
		"/* a */ END":                    true,
		"ROLLBACK":                       true,
		"rollback and chain":             true,
		"ABORT":                          true,
		"PREPARE TRANSACTION 'x'":        true,
		"ROLLBACK TO SAVEPOINT s":        false,
		"rollback to s":                  false,
		"ROLLBACK WORK TO SAVEPOINT s":   false,
		"SAVEPOINT s":                    false,
		"BEGIN":                          false, // only a warning inside a transaction
		"SELECT 'commit'":                false,
		"UPDATE t SET endpoint = 1":      false,
		"CREATE TABLE commits (id int)":  false,
		"COMMENT ON TABLE t IS 'x'":      false,
		"ENDPOINTS":                      false,
		"DO $$ BEGIN COMMIT; END $$":     false, // fails on its own: invalid transaction termination
		"PREPARE q AS SELECT 1":          false,
		"-- COMMIT is refused\nSELECT 1": false,
	} {
		if got := refuseTxControl(stmt) != nil; got != refuse {
			t.Errorf("refuseTxControl(%q) refused = %v, want %v", stmt, got, refuse)
		}
	}
}

func TestRefuseCopyFromStdin(t *testing.T) {
	for stmt, refuse := range map[string]bool{
		"COPY t FROM STDIN":                           true,
		"copy t (a, b)\nfrom stdin with (format csv)": true,
		"-- load\nCOPY t FROM stdin":                  true,
		"COPY t TO STDOUT":                            false,
		"COPY (SELECT 1) TO STDOUT WITH (FORMAT csv)": false,
		"SELECT 'copy from stdin'":                    false,
	} {
		if got := refuseCopyFromStdin(stmt) != nil; got != refuse {
			t.Errorf("refuseCopyFromStdin(%q) refused = %v, want %v", stmt, got, refuse)
		}
	}
}
