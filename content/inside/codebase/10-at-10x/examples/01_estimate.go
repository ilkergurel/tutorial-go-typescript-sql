// LEARN(inside.codebase.at-10x): before changing a design for scale, multiply today's measurements and see which number breaks first.
package main

import "fmt"

// Measured on 2026-11-13 (see the lesson for where each number comes from).
const (
	bundledLessons = 122       // lessons in the last generated bundle
	bundleBytes    = 1_976_584 // the one generated lesson module (split per lesson on 2026-11-14)
	examples       = 324       // examples in that bundle
	serverShare    = 0.76      // share of examples that run on the backend (not ts-browser)
	serverRunSec   = 1.0       // mean durationMs of 59 server examples, compile included
	browserRunSec  = 0.3       // ts-browser runs take 30-400 ms
)

func main() {
	perLessonKB := float64(bundleBytes) / bundledLessons / 1000
	examplesPerLesson := float64(examples) / bundledLessons
	fmt.Printf("per lesson: %.1f KB of bundle, %.2f examples\n\n", perLessonKB, examplesPerLesson)

	fmt.Printf("%-22s %10s %14s %18s\n", "", "lessons", "bundle (MB)", "verify (minutes)")
	for _, row := range []struct {
		label   string
		lessons int
	}{{"last bundle", bundledLessons}, {"curriculum", 229}, {"10x curriculum", 2290}} {
		ex := examplesPerLesson * float64(row.lessons)
		// verify-lessons runs every example twice, one after another.
		seconds := 2 * ex * (serverShare*serverRunSec + (1-serverShare)*browserRunSec)
		fmt.Printf("%-22s %10d %14.1f %18.0f\n", row.label, row.lessons, perLessonKB*float64(row.lessons)/1000, seconds/60)
	}

	// Two run slots: the most runs per minute the backend can admit.
	fmt.Printf("\nrun slots: 2, so at most %.0f server runs per minute at %.1f s each\n", 2*60/serverRunSec, serverRunSec)
}
