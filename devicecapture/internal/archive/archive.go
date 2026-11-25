package archive

import (
	"context"
	"devicecapture/internal/config"
	"devicecapture/internal/logger"
	"devicecapture/internal/postgres/db"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	ds, dbErr := a.queries.GetArchiveTargets(ctx)
	if dbErr != nil {
		return dbErr
	}
	var paths []string
	for _, d := range ds {
		paths = append(paths, d.ImagePath)
	}
	// Deletion path
	if a.conf.ArchivePath == "" {
		err := a.DeleteAll(paths)
		if err != nil {
			return err
		}
		err = a.queries.PurgeOldestTargets(ctx)
		if err != nil {
			return err
		}
		logger.Debug().Str("Archivist", "Run").Msg("Deleted archive targets from the DB")
	} else {
		err := a.MoveAll(ctx, ds, paths)
		if err != nil {
			logger.Error().Str("Archivist", "Run").
				Err(err).
				Str("MoveAll", "failed").Send()
			return err
		}
		logger.Debug().Str("Archivist", "Run").Msg("MoveAll succeeded")
	}
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

// MoveAll moves inPath files & their parent directories -> conf.ArchivePath
func (a *Archivist) MoveAll(ctx context.Context, deviceImgs []db.DeviceImage, inPaths []string) error {
	// Extract directory names from the list of file paths
	// so we can just delete the directories. This reduces
	// filesystem reads and writes
	logger.Debug().Str("archive", "MoveAll").Send()
	dmap := make(map[string]bool)
	for _, fp := range inPaths {
		dp := dirPath(fp)
		_, exists := dmap[dp]
		if !exists {
			dmap[dp] = true
			// ensure we have an equivalent directory in the ArchivePath
			dest := filepath.Join(a.conf.ArchivePath, filepath.Base(dp))
			err := os.MkdirAll(dest, 0755)
			if err != nil {
				return err
			}
		}
	}
	for _, di := range deviceImgs {
		if strings.Contains(di.ImagePath, a.conf.ArchivePath) {
			logger.Info().Str("archive", "MoveAll").
				Int64("skipping", di.ID).
				Msg("is already archived")
			continue
		}
		if di.AnnotatedPath == nil {
			logger.Info().Str("archive", "MoveAll").
				Int64("skipping", di.ID).
				Msg("annoPath was nil")
			continue
		}
		imgDest := strings.ReplaceAll(
			di.ImagePath,
			a.conf.VideoPath,
			a.conf.ArchivePath,
		)

		err := moveFile(di.ImagePath, imgDest)
		if err != nil {
			logger.Error().Str("archive", "MoveAll").
				Err(err).Send()
			return err
		}

		annoPath := *di.AnnotatedPath
		annotatedDest := strings.ReplaceAll(
			annoPath,
			a.conf.VideoPath,
			a.conf.ArchivePath,
		)
		logger.Debug().Str("archive", "MoveAll").
			Str("moved", annoPath).
			Str("to", annotatedDest).Send()
		err = moveFile(annoPath, annotatedDest)
		if err != nil {
			logger.Error().Str("archive", "MoveAll").
				Str("FS", "failed on moveArchived").
				Err(err).Send()
			return err
		}
		err = a.queries.SetArchived(ctx, db.SetArchivedParams{
			ID:            di.ID,
			ImagePath:     imgDest,
			AnnotatedPath: &annotatedDest,
		})
		if err != nil {
			logger.Error().Str("archive", "MoveAll").
				Str("DB", "failed on SetArchived").
				Err(err).Send()
			return err
		} else {
			logger.Debug().Str("archive", "MoveAll").
				Int64("moved", di.ID).
				Str("nextPath", imgDest).
				Str("status", "Success").Send()
		}
	}
	return nil
}

// dirPath extract the absolute path without the filename and extension
// example: dirPath("myroot/my-dir/my-file.txt") // "myroot/my-dir"
func dirPath(fp string) string {
	return filepath.Dir(fp)
}

func moveFile(src string, dst string) error {
	logger.Debug().
		Str("archive", "moveFile").
		Str("src", src).
		Str("dst", dst).
		Msg("Fallback to copy/delete")
	// Try a fast atomic rename first
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Fallback: copy and delete
	logger.Debug().
		Str("archive", "moveFile").
		Msg("Fallback to copy/delete")
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Remove(src)
}
