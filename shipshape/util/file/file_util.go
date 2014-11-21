package file

import (
	"os"
)

// ChangeDir changes into the specified directory.
// It returns the old directory and a function that
// can be defered to change back into it.
func ChangeDir(dir string) (string, func() error, error) {
	orgDir, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}
	return orgDir, func() error { return os.Chdir(orgDir) }, os.Chdir(dir)
}
