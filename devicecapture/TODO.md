## Archiver
Current behavior deletes after 7 days (hardcoded). We need to allow users configure either:
- Delete media older than `ARCHIVE_DAYS` days
- **OR** Move media to `ARCHIVE_DIR` after `ARCHIVE_DAYS`

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
- 


## Deploys
Currently (2026-Oct-25) torn between two approaches:
- minikube
- The current approach

#### The Problem
`deploy.sh` works, but could be improved if we were to use more of a CI/CD flow
Ie. build on a different machine that has more CPU power (desktop?), push images to the
  docker registry (LAN), then trigger a script on the CM5 to pull the images from the registry.

Alternatively, minikube would add some nice features, but the overhead of Kubernetes might not be wise
  on such a resource-limited device (CM5 w/ 8Gb RAM)

