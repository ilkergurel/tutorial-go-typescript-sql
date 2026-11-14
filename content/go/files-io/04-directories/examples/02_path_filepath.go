package main

import (
	"fmt"
	"path"
	"path/filepath"
)

func main() {
	// path: "/"-separated names that aren't OS paths: URLs, embed.FS, io/fs, zip entries.
	fmt.Println(path.Join("/static", "css", "../img", "logo.png"))
	fmt.Println(path.Dir("raporlar/2026/Şubat.csv"), path.Base("/api/v1/users/"), path.Ext("yedek.tar.gz"))

	// filepath: the OS's paths, with "\" on Windows. ToSlash keeps this output portable.
	p := filepath.Join("data", "raporlar", "Şubat.csv")
	rel, err := filepath.Rel("data", p)
	fmt.Println(filepath.ToSlash(p), filepath.ToSlash(rel), err)

	// Join cleans "..", it doesn't confine: a user-supplied name climbs right out.
	fmt.Println(filepath.ToSlash(filepath.Join("uploads", "../../etc/passwd")))

	// IsLocal is the lexical check for names from users, archives and URLs.
	for _, name := range []string{"raporlar/Şubat.csv", "a/../b", "../secret.env", "a/../../b", "/etc/passwd", ""} {
		fmt.Printf("IsLocal(%q) = %t\n", name, filepath.IsLocal(name))
	}

	// Localize turns a "/" path (from a URL or an fs.FS) into an OS path, or refuses.
	for _, name := range []string{"raporlar/Şubat.csv", "../secret.env"} {
		local, err := filepath.Localize(name)
		fmt.Printf("Localize(%q) = %q, %v\n", name, filepath.ToSlash(local), err)
	}
}
