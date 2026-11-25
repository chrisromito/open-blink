## Archiver
Current behavior deletes after 7 days (hardcoded). We need to allow users configure either:
- Delete media older than `ARCHIVE_DAYS` days
- **OR** Move media to `ARCHIVE_DIR` (/mnt/rackdrive/open-blink-static as of 11-Nov-2026) after `ARCHIVE_DAYS`

----------
Parts of the system that will need work:
- devicecapture/internal/archive/archive.go - Use app config to either
  call the existing `DeleteAll` method OR the **new** method `MoveAll`.
  - `MoveAll` updates records via `ArchiveRepo.SetArchived()`
- devicecapture/internal/postgres/repos/archiver - Use `SetArchived` method to update
  the path values for `DeviceImage` instances -> `AnnotatedPath` & `ImagePath` fields
- Somewhere: Add the logic to move files
- Possibly add a DB field to say "this record is archived"
- Static routing needs to use some other URL prefix
  - Determine if we're better off leveraging NGINX static file serving for archived images



