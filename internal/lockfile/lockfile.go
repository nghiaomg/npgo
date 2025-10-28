package lockfile

import (
	"fmt"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

type PackageEntry struct {
	Name      string `yaml:"name"`
	Version   string `yaml:"version"`
	Resolved  string `yaml:"resolved"`
	Integrity string `yaml:"integrity"`
}

type LockFile struct {
	LockfileVersion int            `yaml:"lockfileVersion"`
	Packages        []PackageEntry `yaml:"packages"`
}

func Path(projectDir string) string {
	return filepath.Join(projectDir, ".npgo-lock.yaml")
}

func Load(projectDir string) (*LockFile, error) {
	p := Path(projectDir)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var lf LockFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, err
	}
	return &lf, nil
}

func Save(projectDir string, lf *LockFile) error {
	data, err := yaml.Marshal(lf)
	if err != nil {
		return err
	}
	p := Path(projectDir)
	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}
	return nil
}
