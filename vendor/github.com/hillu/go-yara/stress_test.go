package yara

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"sync"
	"testing"
)

func TestFileWalk(t *testing.T) {
	if os.ExpandEnv("${TEST_WALK}") == "" {
		t.Skip("Set TEST_WALK to enable scanning files from user's HOME with a dummy ruleset.\n" +
			"(Setting -test.timeout may be a good idea for this.)")
	}
	initialPath := os.ExpandEnv("${TEST_WALK_START}")
	if initialPath == "" {
		if u, err := user.Current(); err != nil {
			t.Skip("Could get user's homedir. You can use TEST_WALK_START " +
				"to set an initial path for filepath.Walk()")
		} else {
			initialPath = u.HomeDir
		}
	}
	r, err := Compile("rule test: tag1 tag2 tag3 { meta: foo = 1 bar = \"xxx\" quux = false condition: true }", nil)
	if err != nil {
		t.Fatal(err)
	}
	wg := sync.WaitGroup{}
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			filepath.Walk(initialPath, func(name string, info os.FileInfo, inErr error) error {
				fmt.Printf("[%02d] %s\n", i, name)
				if inErr != nil {
					fmt.Printf("[%02d] Walk to \"%s\": %s\n", i, name, inErr)
					return nil
				}
				if info.IsDir() && info.Mode()&os.ModeSymlink != 0 {
					fmt.Printf("[%02d] Walk to \"%s\": Skipping symlink\n", i, name)
					return filepath.SkipDir
				}
				if !info.Mode().IsRegular() || info.Size() >= 2000000 {
					return nil
				}
				if m, err := r.ScanFile(name, 0, 0); err == nil {
					fmt.Printf("[%02d] Scan \"%s\": %d\n", i, path.Base(name), len(m))
				} else {
					fmt.Printf("[%02d] Scan \"%s\": %s\n", i, path.Base(name), err)
				}
				return nil
			})
			wg.Done()
		}(i)
	}
	wg.Wait()
}
