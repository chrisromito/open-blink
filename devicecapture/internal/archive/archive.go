package archive

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/logger"
	"devicecapture/internal/postgres/db"
	"os"
	"path/filepath"
)

// Archivist Deletes images from the FS and the DB
type Archivist struct {
	conf    *config.Config
	queries *db.Queries
}

func NewArchivist(c *config.Config, q *db.Queries) *Archivist {
	return &Archivist{
		conf:    c,
		queries: q,
	}
}


func (a *Archivist) Run(ctx context.Context) error {
	ds, err := a.queries.GetArchiveTargets(ctx)
	var paths []string
	for _, d := range ds {
		paths = append(paths, d.ImagePath)
	}
	err = a.DeleteAll(paths)
	if err != nil {
		return err
	}
	err = a.queries.PurgeOldestTargets(ctx)
	if err != nil {
		return err
	}
	logger.Debug().Str("Archivist", "Run").Msg("Deleted archive targets from the DB")
	return nil
}

// DeleteAll deletes inPath files and their parent directories from the file system
func (a *Archivist) DeleteAll(inPaths []string) error {
	// Extract directory names from the list of file paths
	// so we can just delete the directories. This reduces
	// filesystem reads and writes
	dmap := make(map[string]bool)
	for _, fp := range inPaths {
		dp := dirPath(fp)
		_, exists := dmap[dp]
		if !exists {
			dmap[dp] = true
		}
	}
	for dp := range dmap {
		err := os.RemoveAll(dp)
		if err != nil {
			logger.Error().Err(err).
				Msgf("failed to delete %s", dp)
		}
	}
	return nil
}

func dirPath(fp string) string {
	return filepath.Dir(fp)
}
