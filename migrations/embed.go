package migrations

import "embed"

// FS keeps the deployment migrations inside the compiled application as well
// as in the Docker init directory. PostgreSQL upgrades no longer depend on a
// brand-new database volume.
//
//go:embed *.sql
var FS embed.FS
